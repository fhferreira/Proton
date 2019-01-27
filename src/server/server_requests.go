package server

import (
	"log"
	"time"

	"github.com/kshvakov/clickhouse"
)

func (server *server) background() {
	for tick := time.Tick(time.Second); ; {
		block := server.block.Copy()
	loop:
		for block.Reserve(); ; {
			select {
			case request := <-server.reqBacklog:
				block.NumRows++
				block.WriteString(0, request.GetHostname())
				block.WriteString(1, request.GetSchema())
				block.WriteInt16(2, int16(request.GetStatus()))
				block.WriteString(3, request.GetServerName())
				block.WriteString(4, request.GetScriptName())
				block.WriteUInt32(5, request.GetRequestCount())
				block.WriteFloat32(6, request.GetRequestTime())
				block.WriteUInt32(7, request.GetDocumentSize())
				block.WriteUInt32(8, request.GetMemoryPeak())
				block.WriteUInt32(9, request.GetMemoryFootprint())
				block.WriteFloat32(10, request.GetRuUtime())
				block.WriteFloat32(11, request.GetRuStime())
				tagName, tagValue := request.tags()
				{
					block.WriteArray(12, clickhouse.Array(tagName))
					block.WriteArray(13, clickhouse.Array(tagValue))
				}
				// timer
				block.WriteArray(14, clickhouse.Array(request.GetTimerHitCount))
				block.WriteArray(15, clickhouse.Array(request.GetTimerValue))
				block.WriteArray(16, clickhouse.Array(request.GetTimerRuStime))
				block.WriteArray(17, clickhouse.Array(request.GetTimerRuUtime))
				// block.WriteArray(18, clickhouse.Array(Array(T))  TagsName @todo add support of Array(Array(T)) to the driver
				// block.WriteArray(19, clickhouse.Array(Array(T))) TagsValue
				block.WriteUInt32(18, request.timestamp)
			case <-tick:
				break loop
			}
		}
		opsReqProcessed.Add(float64(block.NumRows))
		if err := server.writeBlock(insertIntoRequestsSQL, block); err != nil {
			log.Println("request write error: ", err)
		}
	}
}

const (
	insertIntoRequestsSQL = `
	INSERT INTO proton.requests (
		Hostname
		, Schema
		, Status
		, ServerName
		, ScriptName
		, RequestCount
		, RequestTime
		, DocumentSize
		, MemoryPeak
		, MemoryFootprint
		, Utime
		, Stime
		, Tags.Name
		, Tags.Value
		, Timers.HitCount
		, Timers.Value
		, Timers.Utime
		, Timers.Stime
		/*, Timers.TagsName
		, Tiers.TagsValue*/
		, Timestamp
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?,
		?
	)
	`
)
