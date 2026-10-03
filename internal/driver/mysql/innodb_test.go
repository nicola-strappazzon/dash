package mysql

import "testing"

func TestParseInnoDBStatus(t *testing.T) {
	raw := `
2026-10-03 12:00:00 0 INNODB MONITOR OUTPUT
OS WAIT ARRAY INFO: reservation count 12
OS WAIT ARRAY INFO: signal count 34
RW-shared spins 56, rounds 78, OS waits 9
RW-excl spins 10, rounds 11, OS waits 12
TRANSACTIONS
Trx id counter 5678
History list length 4321
1.50 reads/s, 1024 avg bytes/read, 2.50 writes/s, 3.50 log writes/s
Log sequence number 1000
Log flushed up to 900
Last checkpoint at 800
Buffer pool size   100
Free buffers       20
Database pages     70
Modified db pages  5
Buffer pool hit rate 999 / 1000
1.25 reads/s, 0.00 creates/s, 2.25 writes/s
2 queries inside InnoDB, 1 queries in queue
1.00 inserts/s, 2.00 updates/s, 3.00 deletes/s, 4.00 reads/s
`

	status := parseInnoDBStatus(raw)
	if status.Timestamp != "2026-10-03 12:00:00" || status.HistoryLength != 4321 {
		t.Fatalf("unexpected transactions: %#v", status)
	}
	if status.BufferPoolHitRatePct != 99.9 || status.BufferPoolModifiedPages != 5 {
		t.Fatalf("unexpected buffer pool: %#v", status)
	}
	if status.RowReadsPerSec != 4 || status.Semaphores.RWSharedOSWaits != 9 {
		t.Fatalf("unexpected rates or semaphores: %#v", status)
	}
}
