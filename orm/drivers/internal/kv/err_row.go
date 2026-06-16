package kv

type kvErrRow struct{ err error }

func (e kvErrRow) Scan(dest ...any) error { return e.err }
