package asynchflow

func (pc *AsynchFlow) Close() error {

	Ctx.Close()
	return nil
}
