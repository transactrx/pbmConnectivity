package tlspersistedsynch

import "fmt"

func (pc *TLSPersistedSyncConnect) GetStats(statsMap map[string]interface{}) error {

	// get session count using tls context
	sessionCount := Ctx.GetConnectionCount()
	statsMap["sessionCount"] = fmt.Sprintf("%d", sessionCount)
	return nil
}
