package socketpersistedsynch


func (pc *SocketPersistedSyncConnect) Close() error {

	Ctx.Close()
	return nil
}
