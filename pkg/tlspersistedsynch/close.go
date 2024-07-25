package tlspersistedsynch


func (pc *TLSPersistedSyncConnect) Close() error {

	Ctx.Close()
	return nil
}
