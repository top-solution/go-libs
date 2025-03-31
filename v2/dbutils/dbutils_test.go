package dbutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionString(t *testing.T) {
	cases := []struct {
		name           string
		conf           DBConfig
		expectedString string
	}{

		{
			name: "postgres",
			conf: DBConfig{
				Driver:   "postgres",
				Server:   "localhost",
				Port:     5432,
				User:     "user",
				Password: "password",
				DB:       "databasename",
			},
			expectedString: "postgres://user:password@localhost:5432?dbname=databasename&sslmode=disable",
		},

		{
			name: "mssql",
			conf: DBConfig{
				Driver:   "sqlserver",
				Server:   "localhost",
				Port:     1433,
				User:     "user",
				Password: "password",
				DB:       "databasename",
			},
			expectedString: "sqlserver://user:password@localhost:1433?database=databasename",
		},

		{
			name: "mssql with instance",
			conf: DBConfig{
				Driver:   "sqlserver",
				Server:   "MSSQLSERVER",
				Instance: "INSTANCE",
				DB:       "databasename",
			},
			expectedString: "sqlserver://MSSQLSERVER/INSTANCE?database=databasename",
		},

		{
			name: "mssql with instance and user authentication",
			conf: DBConfig{
				Driver:   "sqlserver",
				Server:   "MSSQLSERVER",
				Instance: "INSTANCE",
				DB:       "databasename",
				User:     "user",
				Password: "password",
			},
			expectedString: "sqlserver://user:password@MSSQLSERVER/INSTANCE?database=databasename",
		},

		{
			name: "mssql with instance and special characters",
			conf: DBConfig{
				Driver:   "sqlserver",
				Server:   `MSSQL\/SERVER`,
				Instance: "INS?TANCE",
				DB:       "databa_;:sename",
			},
			expectedString: "sqlserver://MSSQL%5C%2FSERVER/INS%3FTANCE?database=databa_%3B%3Asename",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			connectionString := fromDBConfToConnectionString(tc.conf)
			assert.Equal(t, tc.expectedString, connectionString)
		})
	}
}
