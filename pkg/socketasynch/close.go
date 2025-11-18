package socketasynch

func (pc *SocketAsyncConnect) Close() error {

	pc.Ctx.Close()
	return nil
}
