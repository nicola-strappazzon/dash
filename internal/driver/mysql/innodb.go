package mysql

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type InnoDBSemaphores struct {
	ReservationCount int64
	SignalCount      int64
	RWSharedSpins    int64
	RWSharedOSWaits  int64
	RWExclSpins      int64
	RWExclOSWaits    int64
}

type InnoDBStatus struct {
	Timestamp  string
	Semaphores InnoDBSemaphores

	TrxIDCounter  int64
	HistoryLength int64

	ReadsPerSec     float64
	WritesPerSec    float64
	LogWritesPerSec float64

	LogSequenceNumber int64
	LogFlushedUpTo    int64
	LastCheckpoint    int64

	BufferPoolSizePages     int64
	BufferPoolFreePages     int64
	BufferPoolDatabasePages int64
	BufferPoolModifiedPages int64
	BufferPoolHitRatePct    float64
	BufferPoolReadsPerSec   float64
	BufferPoolWritesPerSec  float64

	QueriesInsideInnoDB int64
	QueriesInQueue      int64
	InsertsPerSec       float64
	UpdatesPerSec       float64
	DeletesPerSec       float64
	RowReadsPerSec      float64
}

var (
	innodbTimestamp     = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \d+ INNODB MONITOR OUTPUT`)
	innodbReservation   = regexp.MustCompile(`OS WAIT ARRAY INFO: reservation count (\d+)`)
	innodbSignal        = regexp.MustCompile(`OS WAIT ARRAY INFO: signal count (\d+)`)
	innodbRWShared      = regexp.MustCompile(`RW-shared spins (\d+), rounds \d+, OS waits (\d+)`)
	innodbRWExcl        = regexp.MustCompile(`RW-excl spins (\d+), rounds \d+, OS waits (\d+)`)
	innodbTrxIDCounter  = regexp.MustCompile(`Trx id counter (\d+)`)
	innodbHistoryLength = regexp.MustCompile(`History list length (\d+)`)
	innodbFileIORate    = regexp.MustCompile(`(\d+\.?\d*) reads/s, \d+ avg bytes/read, (\d+\.?\d*) writes/s, (\d+\.?\d*) log writes/s`)
	innodbLogSeq        = regexp.MustCompile(`Log sequence number (\d+)`)
	innodbLogFlushed    = regexp.MustCompile(`Log flushed up to\s+(\d+)`)
	innodbCheckpoint    = regexp.MustCompile(`Last checkpoint at\s+(\d+)`)
	innodbBPSize        = regexp.MustCompile(`Buffer pool size\s+(\d+)`)
	innodbBPFree        = regexp.MustCompile(`Free buffers\s+(\d+)`)
	innodbBPDatabase    = regexp.MustCompile(`Database pages\s+(\d+)`)
	innodbBPModified    = regexp.MustCompile(`Modified db pages\s+(\d+)`)
	innodbBPHitRate     = regexp.MustCompile(`Buffer pool hit rate (\d+) / (\d+)`)
	innodbBPRWRate      = regexp.MustCompile(`(\d+\.?\d*) reads/s, (\d+\.?\d*) creates/s, (\d+\.?\d*) writes/s`)
	innodbQueriesInside = regexp.MustCompile(`(\d+) queries inside InnoDB, (\d+) queries in queue`)
	innodbRowOpsRate    = regexp.MustCompile(`(\d+\.?\d*) inserts/s, (\d+\.?\d*) updates/s, (\d+\.?\d*) deletes/s, (\d+\.?\d*) reads/s`)
)

// InnoDB reads and parses SHOW ENGINE INNODB STATUS.
func (m *MySQL) InnoDB(ctx context.Context) (InnoDBStatus, error) {
	var engineType, name, raw string
	if err := m.db.QueryRowContext(ctx, "SHOW ENGINE INNODB STATUS").Scan(&engineType, &name, &raw); err != nil {
		return InnoDBStatus{}, fmt.Errorf("reading InnoDB status: %w", err)
	}
	return parseInnoDBStatus(raw), nil
}

func parseInnoDBStatus(raw string) InnoDBStatus {
	var out InnoDBStatus
	if match := innodbTimestamp.FindStringSubmatch(raw); match != nil {
		out.Timestamp = match[1]
	}
	if match := innodbReservation.FindStringSubmatch(raw); match != nil {
		out.Semaphores.ReservationCount = parseInnoDBInt(match[1])
	}
	if match := innodbSignal.FindStringSubmatch(raw); match != nil {
		out.Semaphores.SignalCount = parseInnoDBInt(match[1])
	}
	if match := innodbRWShared.FindStringSubmatch(raw); match != nil {
		out.Semaphores.RWSharedSpins = parseInnoDBInt(match[1])
		out.Semaphores.RWSharedOSWaits = parseInnoDBInt(match[2])
	}
	if match := innodbRWExcl.FindStringSubmatch(raw); match != nil {
		out.Semaphores.RWExclSpins = parseInnoDBInt(match[1])
		out.Semaphores.RWExclOSWaits = parseInnoDBInt(match[2])
	}

	if match := innodbTrxIDCounter.FindStringSubmatch(raw); match != nil {
		out.TrxIDCounter = parseInnoDBInt(match[1])
	}
	if match := innodbHistoryLength.FindStringSubmatch(raw); match != nil {
		out.HistoryLength = parseInnoDBInt(match[1])
	}
	if match := innodbFileIORate.FindStringSubmatch(raw); match != nil {
		out.ReadsPerSec, out.WritesPerSec, out.LogWritesPerSec = parseInnoDBFloat(match[1]), parseInnoDBFloat(match[2]), parseInnoDBFloat(match[3])
	}
	if match := innodbLogSeq.FindStringSubmatch(raw); match != nil {
		out.LogSequenceNumber = parseInnoDBInt(match[1])
	}
	if match := innodbLogFlushed.FindStringSubmatch(raw); match != nil {
		out.LogFlushedUpTo = parseInnoDBInt(match[1])
	}
	if match := innodbCheckpoint.FindStringSubmatch(raw); match != nil {
		out.LastCheckpoint = parseInnoDBInt(match[1])
	}
	if match := innodbBPSize.FindStringSubmatch(raw); match != nil {
		out.BufferPoolSizePages = parseInnoDBInt(match[1])
	}
	if match := innodbBPFree.FindStringSubmatch(raw); match != nil {
		out.BufferPoolFreePages = parseInnoDBInt(match[1])
	}
	if match := innodbBPDatabase.FindStringSubmatch(raw); match != nil {
		out.BufferPoolDatabasePages = parseInnoDBInt(match[1])
	}
	if match := innodbBPModified.FindStringSubmatch(raw); match != nil {
		out.BufferPoolModifiedPages = parseInnoDBInt(match[1])
	}
	if match := innodbBPHitRate.FindStringSubmatch(raw); match != nil {
		if total := parseInnoDBInt(match[2]); total > 0 {
			out.BufferPoolHitRatePct = float64(parseInnoDBInt(match[1])) / float64(total) * 100
		}
	}
	if match := innodbBPRWRate.FindStringSubmatch(raw); match != nil {
		out.BufferPoolReadsPerSec, out.BufferPoolWritesPerSec = parseInnoDBFloat(match[1]), parseInnoDBFloat(match[3])
	}
	if match := innodbQueriesInside.FindStringSubmatch(raw); match != nil {
		out.QueriesInsideInnoDB, out.QueriesInQueue = parseInnoDBInt(match[1]), parseInnoDBInt(match[2])
	}
	if match := innodbRowOpsRate.FindStringSubmatch(raw); match != nil {
		out.InsertsPerSec, out.UpdatesPerSec, out.DeletesPerSec, out.RowReadsPerSec = parseInnoDBFloat(match[1]), parseInnoDBFloat(match[2]), parseInnoDBFloat(match[3]), parseInnoDBFloat(match[4])
	}
	return out
}

func parseInnoDBInt(value string) int64 {
	number, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return number
}
func parseInnoDBFloat(value string) float64 {
	number, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return number
}
