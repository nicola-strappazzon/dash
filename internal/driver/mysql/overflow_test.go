package mysql

import "testing"

func TestAutoIncrementCapacity(t *testing.T) {
	tests := []struct {
		columnType string
		maximum    uint64
	}{
		{"tinyint", 127},
		{"tinyint unsigned", 255},
		{"smallint", 32767},
		{"smallint unsigned", 65535},
		{"mediumint", 8388607},
		{"mediumint unsigned", 16777215},
		{"int", 2147483647},
		{"int unsigned", 4294967295},
		{"bigint", 9223372036854775807},
		{"bigint unsigned", 18446744073709551615},
	}

	for _, test := range tests {
		t.Run(test.columnType, func(t *testing.T) {
			if got := autoIncrementMax(test.columnType); got != test.maximum {
				t.Errorf("autoIncrementMax(%q) = %d, want %d", test.columnType, got, test.maximum)
			}
		})
	}
}

func TestAutoIncrementCalculateCapacity(t *testing.T) {
	column := AutoIncrement{ColumnType: "smallint unsigned", CurrentValue: 16384}
	column.calculateCapacity()

	if column.MaxValue != 65535 {
		t.Errorf("MaxValue = %d, want 65535", column.MaxValue)
	}
	if column.Remaining != 49151 {
		t.Errorf("Remaining = %d, want 49151", column.Remaining)
	}
	if column.UsagePercent < 24.9 || column.UsagePercent > 25.1 {
		t.Errorf("UsagePercent = %f, want about 25", column.UsagePercent)
	}
}
