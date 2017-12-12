package squirrel

import (
	"fmt"
	"io"
)

type part struct {
	pred interface{}
	args []interface{}
}

func newPart(pred interface{}, args ...interface{}) Sqlizer {
	return &part{pred, args}
}

func (p part) ToSql() (sql string, args []interface{}) {
	switch pred := p.pred.(type) {
	case nil:
		// no-op
	case Sqlizer:
		sql, args= pred.ToSql()
	case string:
		sql = pred
		args = p.args
	default:
		panic(fmt.Errorf("expected string or Sqlizer, not %T", pred))
	}
	return
}

func appendToSql(parts []Sqlizer, w io.Writer, sep string, args []interface{}) ([]interface{}, error) {
	length := len(parts)
	for i, p := range parts {
		partSql, partArgs := p.ToSql()
		 if len(partSql) == 0 {
			continue
		}

		_, err := io.WriteString(w, partSql)
		if err != nil {
			return nil, err
		}

		if i < length-1 {
			_, err := io.WriteString(w, sep)
			if err != nil {
				return nil, err
			}
		}

		args = append(args, partArgs...)
	}
	return args, nil
}
