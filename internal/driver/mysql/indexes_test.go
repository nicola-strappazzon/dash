package mysql

import "testing"

func TestFormatIndexColumns(t *testing.T) {
	columns := []IndexColumn{
		{Position: 1, Name: "account_id", Collation: "A"},
		{Position: 2, Name: "email", Collation: "D", SubPart: 12, Nullable: true},
		{Position: 3, Expression: "lower(`name`)", Collation: "A"},
	}

	if got, want := FormatIndexColumns(columns), "1:account_id ASC NOT NULL, 2:email(12) DESC NULL, 3:(lower(`name`)) ASC NOT NULL"; got != want {
		t.Errorf("FormatIndexColumns() = %q, want %q", got, want)
	}
}
