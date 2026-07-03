package main

import "testing"

func TestValidateSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sql     string
		mode    SQLMode
		wantErr bool
	}{
		{name: "query select", sql: "select 1", mode: SQLModeQuery},
		{name: "query select into", sql: "select * into #tmp from users", mode: SQLModeQuery, wantErr: true},
		{name: "query multi statement", sql: "select 1; select 2", mode: SQLModeQuery, wantErr: true},
		{name: "write insert", sql: "insert into demo(id) values (1)", mode: SQLModeWrite},
		{name: "write select rejected", sql: "select 1", mode: SQLModeWrite, wantErr: true},
		{name: "write create rejected", sql: "create table demo(id int)", mode: SQLModeWrite, wantErr: true},
		{name: "migrate create table", sql: "create table demo(id int)", mode: SQLModeMigrate},
		{name: "migrate multi statement", sql: "create table t1(id int); create table t2(id int)", mode: SQLModeMigrate},
		{name: "migrate with ddl", sql: "alter table users add column age int", mode: SQLModeMigrate},
		{name: "migrate with dml", sql: "insert into users(name) values ('test')", mode: SQLModeMigrate},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSQL(tt.sql, tt.mode)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateSQL() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateSQL() error = %v, want nil", err)
			}
		})
	}
}
