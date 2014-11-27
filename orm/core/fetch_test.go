// Copyright 2014 by caixw, All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package core

import (
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/caixw/lib.go/assert"
	_ "github.com/mattn/go-sqlite3"
)

type FetchUser struct {
	Id       int    `orm:"name(id);ai(1,2);"`
	Email    string `orm:"unique(unique_index);nullable;pk(pk_name)"`
	Username string `orm:"index(index)"`
	Group    int    `orm:"name(group);fk(fk_group,group,id)"`

	Regdate int `orm:"-"`
}

func TestParseObj(t *testing.T) {
	a := assert.New(t)
	obj := &FetchUser{Id: 5}
	mapped := map[string]reflect.Value{}

	v := reflect.ValueOf(obj).Elem()
	a.True(v.IsValid())

	err := parseObj(v, &mapped)
	a.NotError(err).Equal(4, len(mapped))

	// 忽略的字段
	_, found := mapped["Regdate"]
	a.False(found)

	// 判断字段是否存在
	vi, found := mapped["id"]
	a.True(found).True(vi.IsValid())

	// 设置字段的值
	mapped["id"].Set(reflect.ValueOf(36))
	a.Equal(36, obj.Id)
	mapped["Email"].SetString("email")
	a.Equal("email", obj.Email)
	mapped["Username"].SetString("username")
	a.Equal("username", obj.Username)
	mapped["group"].SetInt(1)
	a.Equal(1, obj.Group)
}

func initDB(a *assert.Assertion) *sql.DB {
	db, err := sql.Open("sqlite3", "./test.db")
	a.NotError(err).NotNil(db)

	/* 创建表 */
	sql := `create table user (
        id integer not null primary key, 
        Email text,
        Username text,
        [group] interger)`
	_, err = db.Exec(sql)
	a.NotError(err)

	/* 插入数据 */
	tx, err := db.Begin()
	a.NotError(err).NotNil(tx)

	stmt, err := tx.Prepare("insert into user(id, Email,Username,[group]) values(?, ?, ?, ?)")
	a.NotError(err).NotNil(stmt)

	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("email-%d", i), fmt.Sprintf("username-%d", i), 1)
		a.NotError(err)
	}
	tx.Commit()
	stmt.Close()

	return db
}

func closeDB(db *sql.DB, a *assert.Assertion) {
	db.Close()
	a.NotError(os.Remove("./test.db"))
	a.FileNotExists("./test.db")
}

func TestFetch2Objs(t *testing.T) {
	a := assert.New(t)
	db := initDB(a)
	defer closeDB(db, a)

	sql := `SELECT id,Email FROM user WHERE id<2 ORDER BY id`
	rows, err := db.Query(sql)
	a.NotError(err).NotNil(rows)
	objs := []*FetchUser{
		&FetchUser{},
		&FetchUser{},
	}

	a.NotError(Fetch2Objs(objs, rows))

	a.Equal([]*FetchUser{
		&FetchUser{Id: 0, Email: "email-0"},
		&FetchUser{Id: 1, Email: "email-1"},
	}, objs)
}

func TestFetch2Maps(t *testing.T) {
	a := assert.New(t)
	db := initDB(a)
	defer closeDB(db, a)

	sql := `SELECT id,Email FROM user WHERE id<2 ORDER BY id`
	rows, err := db.Query(sql)
	a.NotError(err).NotNil(rows)

	mapped, err := Fetch2Maps(false, rows)
	a.NotError(err).NotNil(mapped)

	a.Equal([]map[string]interface{}{
		map[string]interface{}{"id": 0, "Email": "email-0"},
		map[string]interface{}{"id": 1, "Email": "email-1"},
	}, mapped)
}

func TestFetch2MapsString(t *testing.T) {
	a := assert.New(t)
	db := initDB(a)
	defer closeDB(db, a)

	sql := `SELECT id,Email FROM user WHERE id<2 ORDER BY id`
	rows, err := db.Query(sql)
	a.NotError(err).NotNil(rows)

	mapped, err := Fetch2MapsString(false, rows)
	a.NotError(err).NotNil(mapped)

	a.Equal(mapped, []map[string]string{
		map[string]string{"id": "0", "Email": "email-0"},
		map[string]string{"id": "1", "Email": "email-1"},
	})
}

func TestFetchColumns(t *testing.T) {
	a := assert.New(t)
	db := initDB(a)
	defer closeDB(db, a)

	sql := `SELECT id FROM user WHERE id<2 ORDER BY id`
	rows, err := db.Query(sql)
	a.NotError(err).NotNil(rows)

	cols, err := FetchColumns(false, "id", rows)
	a.NotError(err).NotNil(cols)

	a.Equal([]interface{}{0, 1}, cols)
}

func TestFetchColumnsString(t *testing.T) {
	a := assert.New(t)
	db := initDB(a)
	defer closeDB(db, a)

	sql := `SELECT id FROM user WHERE id<2 ORDER BY id`
	rows, err := db.Query(sql)
	a.NotError(err).NotNil(rows)

	cols, err := FetchColumnsString(false, "id", rows)
	a.NotError(err).NotNil(cols)

	a.Equal([]string{"0", "1"}, cols)
}