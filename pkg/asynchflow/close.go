package asynchflow

func (pc *AsynchFlow) Close() error {

	pc.Ctx.Close()
	return nil
}
