// Copyright 2014 by caixw, All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"unicode"

	"github.com/caixw/lib.go/encoding/tag"
)

// 将v分解成map[string]reflect.Value形式，其中键名为对象的字段名，
// 键值为字段的值，支持匿名字段，不会导出不可导出(小写字母开头)的
// 字段，也不会导出struct tag以-开头的字段。
//
// 假定v.Kind()==reflect.Struct，不再进行判断
func parseObj(v reflect.Value, ret map[string]reflect.Value) {
	vt := v.Type()
	num := vt.NumField()
	for i := 0; i < num; i++ {
		field := vt.Field(i)

		if field.Anonymous { // 匿名对象
			parseObj(v.Field(i), ret)
		}

		tagTxt := field.Tag.Get("orm")
		if len(tagTxt) == 0 { // 不存在Struct tag
			if unicode.IsUpper(rune(field.Name[0])) {
				ret[field.Name] = v.Field(i)
			}
		}

		if tagTxt[0] == '-' { // 该字段被标记为忽略
			continue
		}

		name, found := tag.Get(tagTxt, "name")
		if !found { // 没有指定name属性，继续使用字段名
			if unicode.IsUpper(rune(field.Name[0])) {
				ret[field.Name] = v.Field(i)
			}
		}
		ret[name[0]] = v.Field(i)
	} // end for
}

// 读取rows中的一行数据，并保存到v中。其中v必须为一个struct类型。
func fetch2Obj(v reflect.Value, rows *sql.Rows) error {
	if !rows.Next() { // 空行
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// 从rows中导出的数据，暂时保存在此
	buf := make([]interface{}, len(cols), len(cols))
	for i := 0; i < len(cols); i++ {
		var v interface{}
		buf[i] = &v
	}

	if err := rows.Scan(buf...); err != nil {
		return err
	}

	items := make(map[string]reflect.Value, 0)
	parseObj(v, items)

	for index, colName := range cols {
		if item, found := items[colName]; found {
			item.Set(reflect.ValueOf(buf[index]))
		}
	}

	return nil
}

func Fetch2Objs(obj interface{}, rows *sql.Rows) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice:
		itemType := val.Type().Elem()
		if itemType.Kind() != reflect.Struct {
			return fmt.Errorf("元素类型只能为reflect.Struct，当前为[%v]", itemType.Kind())
		}

		v := reflect.New(itemType)
		switch err := fetch2Obj(v, rows); err {
		case nil:
			val.Set(reflect.Append(val, v))
		case sql.ErrNoRows:
			return nil
		default:
			return err
		}
	case reflect.Struct:
		if err := fetch2Obj(val, rows); err != nil {
			return err
		}
	default:
		return errors.New("只支持Slice和Struct指针")
	}

	return nil
}

// 将rows中的数据导出到map中
//
// once 是否只查询一条记录，若为true，则返回长度为1的slice
// rows 查询的结果
// 对外公开，方便db包调用
func Fetch2Maps(once bool, rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 临时缓存，用于保存从rows中读取出来的一行。
	buff := make([]interface{}, len(cols))
	for i, _ := range cols {
		var value interface{}
		buff[i] = &value
	}

	var data []map[string]interface{}
	for rows.Next() {
		if err := rows.Scan(buff...); err != nil {
			return nil, err
		}

		line := make(map[string]interface{}, len(cols))
		for i, v := range cols {
			if buff[i] == nil {
				continue
			}
			value := reflect.Indirect(reflect.ValueOf(buff[i]))
			line[v] = value.Interface()
		}

		data = append(data, line)
		if once {
			return data, nil
		}
	}

	return data, nil
}

// 导出某列的所有数据
//
// colName 该列的名称，若不指定了不存在的名称，返回error
func FetchColumns(once bool, colName string, rows *sql.Rows) ([]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	index := -1 // 该列名在所rows.Columns()中的索引号
	buff := make([]interface{}, len(cols))
	for i, v := range cols {
		var value interface{}
		buff[i] = &value

		if colName == v { // 获取index的值
			index = i
		}
	}

	if index == -1 {
		return nil, errors.New("指定的名不存在")
	}

	var data []interface{}
	for rows.Next() {
		if err := rows.Scan(buff...); err != nil {
			return nil, err
		}
		data = append(data, buff[index])
		if once {
			return data, nil
		}
	}

	return data, nil
}
