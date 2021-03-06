package server

import (
	"database/sql"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse-Ninja/Proton/proto/pinba"
	"github.com/kshvakov/clickhouse"
	"github.com/kshvakov/clickhouse/lib/data"
)

func RunServer(options Options) error {
	conn, err := net.ListenPacket("udp", options.Address)
	if err != nil {
		return err
	}
	log.Printf("Proton server listen UDP [%s], Prometheus exporter [%s] concurrency: %d", options.Address, options.MetricsAddress, options.Concurrency)
	server := server{
		dsn:         options.DSN,
		reqBacklog:  make(chan request, options.BacklogSize),
		dictBacklog: make(chan dict, 1000),
		connections: make(chan clickhouse.Clickhouse, options.Concurrency),
	}
	if server.sqlConnection, err = sql.Open("clickhouse", options.DSN); err != nil {
		return err
	}
	server.sqlConnection.SetMaxOpenConns(2)
	server.sqlConnection.SetConnMaxLifetime(time.Hour)
	if server.block, err = server.prepareBlock(insertIntoRequestsSQL); err != nil {
		return err
	}
	if server.dictBlock, err = server.prepareBlock(insertIntoDictionarySQL); err != nil {
		return err
	}
	cntConcurrency.Set(float64(options.Concurrency))
	go server.metrics(options.MetricsAddress)
	go server.backgroundDictionary()
	for i := 0; i < options.Concurrency; i++ {
		go server.listen(conn)
		go server.background()
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	{
		log.Printf("stopped signal[%s]", <-signals)
	}
	return nil
}

type server struct {
	dsn           string
	block         *data.Block
	dictBlock     *data.Block
	reqBacklog    chan request
	dictBacklog   chan dict
	connections   chan clickhouse.Clickhouse
	sqlConnection *sql.DB
}

func (server *server) prepareBlock(sql string) (block *data.Block, _ error) {
	conn, err := server.connection()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.Begin()
	if _, err = conn.Prepare(sql); err != nil {
		return nil, err
	}
	if block, err = conn.Block(); err != nil {
		return nil, err
	}
	return block, nil
}

func (server *server) writeBlock(sql string, block *data.Block) error {
	if block.NumRows == 0 {
		return nil
	}
	conn, err := server.connection()
	if err != nil {
		return err
	}
	conn.Begin()
	if _, err = conn.Prepare(sql); err != nil {
		return server.releaseConnection(conn, err)
	}
	if err = conn.WriteBlock(block); err != nil {
		return server.releaseConnection(conn, err)
	}
	return server.releaseConnection(conn, conn.Commit())
}

func (server *server) connection() (clickhouse.Clickhouse, error) {
	select {
	case conn := <-server.connections:
		return conn, nil
	default:
		return clickhouse.OpenDirect(server.dsn)
	}
}

func (server *server) releaseConnection(conn clickhouse.Clickhouse, err error) error {
	if err == nil {
		select {
		case server.connections <- conn:
			return nil
		default:
		}
	}
	conn.Close()
	return err
}

func (server *server) listen(conn net.PacketConn) {
	var (
		buffer [math.MaxUint16]byte
		dictID = make(map[uint64]struct{}, 1000)
	)
	for {
		var req pinba.Request
		if ln, _, err := conn.ReadFrom(buffer[:]); err == nil {
			if err := req.Unmarshal(buffer[:ln]); err == nil {
				container := request{
					Request:   req,
					timestamp: now(),
				}
				select {
				case server.reqBacklog <- container:
				default:
					log.Println("backlog is full")
				}
				dictionary := [][]string{
					{"Schema", container.GetSchema()},
					{"Hostname", container.GetHostname()},
					{"ServerName", container.GetServerName()},
					{"ScriptName", container.GetScriptName()},
				}
				for _, value := range container.Dictionary {
					dictionary = append(dictionary, []string{"Dictionary", value})
				}
				for _, tuple := range dictionary {
					id := cityHash64(tuple[1])
					if _, exists := dictID[id]; !exists {
						dictID[id] = struct{}{}
						select {
						case server.dictBacklog <- dict{
							id:     id,
							value:  tuple[1],
							column: tuple[0],
						}:
						default:
						}
					}
				}
			}
		}
	}
}
