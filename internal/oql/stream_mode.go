package oql

type StreamMode int

const (
	StreamPassthrough StreamMode = iota
	StreamBlocking
	StreamWindowed
	StreamBounded
)

func CommandStreamMode(cmd Command) StreamMode {
	switch cmd.(type) {
	case *WhereCommand, *EvalCommand, *RenameCommand, *FieldsCommand,
		*RexCommand, *FillNullCommand, *LookupCommand, *MvExpandCommand:
		return StreamPassthrough
	case *StatsCommand, *TopCommand, *RareCommand, *SortCommand,
		*DedupCommand, *TailCommand, *JoinCommand:
		return StreamBlocking
	case *HeadCommand:
		return StreamBounded
	case *TimeChartCommand, *TransactionCommand:
		return StreamWindowed
	default:
		return StreamBlocking
	}
}
