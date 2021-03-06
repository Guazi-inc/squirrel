package squirrel

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/lann/builder"
)

type insertData struct {
	PlaceholderFormat PlaceholderFormat
	Prefixes          exprs
	Options           []string
	Into              string
	Columns           []string
	Values            [][]interface{}
	Suffixes          exprs
	Select            *SelectBuilder
}

func (d *insertData) ToSql() (sqlStr string, args []interface{}) {
	var err error
	if len(d.Into) == 0 {
		err = errors.New("insert statements must specify a table")
		panic(err)
		return
	}
	if len(d.Values) == 0 && d.Select == nil {
		err = errors.New("insert statements must have at least one set of values or select clause")
		panic(err)
		return
	}

	sql := &bytes.Buffer{}

	if len(d.Prefixes) > 0 {
		args, _ = d.Prefixes.AppendToSql(sql, " ", args)
		sql.WriteString(" ")
	}

	sql.WriteString("INSERT ")

	if len(d.Options) > 0 {
		sql.WriteString(strings.Join(d.Options, " "))
		sql.WriteString(" ")
	}

	sql.WriteString("INTO ")
	sql.WriteString(d.Into)
	sql.WriteString(" ")

	if len(d.Columns) > 0 {
		sql.WriteString("(")
		sql.WriteString(strings.Join(d.Columns, ","))
		sql.WriteString(") ")
	}

	if d.Select != nil {
		args, err = d.appendSelectToSQL(sql, args)
		if err != nil{
			panic(err)
		}
	} else {
		args, err = d.appendValuesToSQL(sql, args)
		if err != nil {
			return
		}
	}
	if len(d.Suffixes) > 0 {
		sql.WriteString(" ")
		args, _ = d.Suffixes.AppendToSql(sql, " ", args)
	}

	sqlStr, err = d.PlaceholderFormat.ReplacePlaceholders(sql.String())
	if err != nil {
		panic(err)
	}
	return
}

func (d *insertData) appendValuesToSQL(w io.Writer, args []interface{}) ([]interface{}, error) {
	if len(d.Values) == 0 {
		return args, errors.New("values for insert statements are not set")
	}

	io.WriteString(w, "VALUES ")

	valuesStrings := make([]string, len(d.Values))
	for r, row := range d.Values {
		valueStrings := make([]string, len(row))
		for v, val := range row {
			e, isExpr := val.(expr)
			if isExpr {
				valueStrings[v] = e.sql
				args = append(args, e.args...)
			} else {
				valueStrings[v] = "?"
				args = append(args, val)
			}
		}
		valuesStrings[r] = fmt.Sprintf("(%s)", strings.Join(valueStrings, ","))
	}

	io.WriteString(w, strings.Join(valuesStrings, ","))

	return args, nil
}

func (d *insertData) appendSelectToSQL(w io.Writer, args []interface{}) ([]interface{}, error) {
	if d.Select == nil {
		return args, errors.New("select clause for insert statements are not set")
	}

	selectClause, sArgs:= d.Select.ToSql()

	io.WriteString(w, selectClause)
	args = append(args, sArgs...)

	return args, nil
}

// Builder

// InsertBuilder builds SQL INSERT statements.
type InsertBuilder builder.Builder

func init() {
	builder.Register(InsertBuilder{}, insertData{})
}

// Format methods

// PlaceholderFormat sets PlaceholderFormat (e.g. Question or Dollar) for the
// query.
func (b InsertBuilder) PlaceholderFormat(f PlaceholderFormat) InsertCondition {
	return builder.Set(b, "PlaceholderFormat", f).(InsertBuilder)
}

// SQL methods

// ToSql builds the query into a SQL string and bound args.
func (b InsertBuilder) ToSql() (string, []interface{}) {
	data := builder.GetStruct(b).(insertData)
	return data.ToSql()
}

// Prefix adds an expression to the beginning of the query
func (b InsertBuilder) Prefix(sql string, args ...interface{}) InsertCondition {
	return builder.Append(b, "Prefixes", Expr(sql, args...)).(InsertBuilder)
}

// Options adds keyword options before the INTO clause of the query.
func (b InsertBuilder) Options(options ...string) InsertCondition {
	return builder.Extend(b, "Options", options).(InsertBuilder)
}

// Into sets the INTO clause of the query.
func (b InsertBuilder) Into(from string) InsertCondition {
	return builder.Set(b, "Into", from).(InsertBuilder)
}

// Columns adds insert columns to the query.
func (b InsertBuilder) Columns(columns ...string) InsertCondition {
	return builder.Extend(b, "Columns", columns).(InsertBuilder)
}

// Values adds a single row's values to the query.
func (b InsertBuilder) Values(values ...interface{}) InsertCondition {
	return builder.Append(b, "Values", values).(InsertBuilder)
}

// Suffix adds an expression to the end of the query
func (b InsertBuilder) Suffix(sql string, args ...interface{}) InsertCondition {
	return builder.Append(b, "Suffixes", Expr(sql, args...)).(InsertBuilder)
}

// SetMap set columns and values for insert builder from a map of column name and value
// note that it will reset all previous columns and values was set if any
func (b InsertBuilder) SetMap(clauses map[string]interface{}) InsertCondition {
	cols := make([]string, 0, len(clauses))
	vals := make([]interface{}, 0, len(clauses))
	for col, val := range clauses {
		cols = append(cols, col)
		vals = append(vals, val)
	}

	b = builder.Set(b, "Columns", cols).(InsertBuilder)
	b = builder.Set(b, "Values", [][]interface{}{vals}).(InsertBuilder)
	return b
}

// Select set Select clause for insert query
// If Values and Select are used, then Select has higher priority
func (b InsertBuilder) Select(sb SelectCondition) InsertCondition {
	return builder.Set(b, "Select", &sb).(InsertBuilder)
}
