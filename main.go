package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strings"
)

const (
	DB_USER     = "odc"
	DB_PASSWORD = "nHVMdeU2Dx2sbzGD"
	DB_NAME     = "odc2"
	// PKG_NAME gives name of the package using the stucts
	PKG_NAME = "DbStructs"
	// TAG_LABEL produces tags commonly used to match database field names with Go struct members
	TAG_LABEL = "db"
)

type ColumnSchema struct {
	TableName              string
	ColumnName             string
	IsNullable             string
	DataType               string
	CharacterMaximumLength sql.NullInt64
	NumericPrecision       sql.NullInt64
	NumericScale           sql.NullInt64
	ColumnType             string
	ColumnKey              string
}

func writeStructs(schemas []ColumnSchema) (int, error) {
	file, err := os.Create("db_structs.go")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	currentTable := ""

	neededImports := make(map[string]bool)

	// First, get body text into var out
	out := ""
	for _, cs := range schemas {

		if cs.TableName != currentTable {
			if currentTable != "" {
				out = out + "}\n\n"
			}
			out = out + "type " + formatName(cs.TableName) + " struct{\n"
		}

		goType, requiredImport, err := goType(&cs)
		if requiredImport != "" {
			neededImports[requiredImport] = true
		}

		if err != nil {
			log.Fatal(err)
		}
		out = out + "\t" + formatName(cs.ColumnName) + " " + goType
		if len(TAG_LABEL) > 0 {
			out = out + "\t`" + TAG_LABEL + ":\"" + cs.ColumnName + "\"`"
		}
		out = out + "\n"
		currentTable = cs.TableName

	}
	out = out + "}"

	// Now add the header section
	header := "package " + PKG_NAME + "\n\n"
	if len(neededImports) > 0 {
		header = header + "import (\n"
		for imp := range neededImports {
			header = header + "\t\"" + imp + "\"\n"
		}
		header = header + ")\n\n"
	}

	totalBytes, err := fmt.Fprint(file, header+out)
	if err != nil {
		log.Fatal(err)
	}
	return totalBytes, nil
}

func getSchema() []ColumnSchema {
	conn, err := sql.Open("mysql", DB_USER+":"+DB_PASSWORD+"@/information_schema")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	q := "SELECT TABLE_NAME, COLUMN_NAME, IS_NULLABLE, DATA_TYPE, " +
		"CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE, COLUMN_TYPE, " +
		"COLUMN_KEY FROM COLUMNS WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME, ORDINAL_POSITION"
	rows, err := conn.Query(q, DB_NAME)
	if err != nil {
		log.Fatal(err)
	}
	columns := []ColumnSchema{}
	for rows.Next() {
		cs := ColumnSchema{}
		err := rows.Scan(&cs.TableName, &cs.ColumnName, &cs.IsNullable, &cs.DataType,
			&cs.CharacterMaximumLength, &cs.NumericPrecision, &cs.NumericScale,
			&cs.ColumnType, &cs.ColumnKey)
		if err != nil {
			log.Fatal(err)
		}
		columns = append(columns, cs)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return columns
}

func formatName(name string) string {
	parts := strings.Split(name, "_")
	newName := ""
	for _, p := range parts {
		newName = newName + strings.Replace(p, string(p[0]), strings.ToUpper(string(p[0])), 1)
	}
	return newName
}

func goType(col *ColumnSchema) (string, string, error) {
	requiredImport := ""
	if col.IsNullable == "YES" {
		requiredImport = "database/sql"
	}
	var gt string = ""
	switch col.DataType {
	case "varchar", "enum", "text", "longtext", "mediumtext":
		if col.IsNullable == "YES" {
			gt = "sql.NullString"
		} else {
			gt = "string"
		}
	case "date", "time", "datetime", "timestamp":
		gt, requiredImport = "time.Time", "time"
	case "tinyint", "smallint", "int", "mediumint", "bigint":
		if col.IsNullable == "YES" {
			gt = "sql.NullInt64"
		} else {
			gt = "int64"
		}
	case "float", "decimal", "double":
		if col.IsNullable == "YES" {
			gt = "sql.NullFloat64"
		} else {
			gt = "float64"
		}
	}
	if gt == "" {
		n := col.TableName + "." + col.ColumnName
		return "", "", errors.New("No compatible datatype for " + n + " found")
	}
	return gt, requiredImport, nil
}

func main() {
	columns := getSchema()
	bytes, err := writeStructs(columns)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Ok %d\n", bytes)
}
