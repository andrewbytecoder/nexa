package tcpterm

import (
	"errors"
	"fmt"
	"github.com/google/gopacket/pcap"
	"github.com/nexa/pkg/ctx"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/google/gopacket"
	"github.com/rivo/tview"
)

const (
	TailMode = iota
	SelectMode
)

type TcpTerm struct {
	ctx    *ctx.Ctx
	logger *zap.Logger

	src        *gopacket.PacketSource
	view       *tview.Application
	primitives []tview.Primitive
	table      *tview.Table
	detail     *tview.TextView
	dump       *tview.TextView
	frame      *tview.Frame
	packets    []gopacket.Packet
	mode       int

	handle *pcap.Handle

	// Command line flags.
	iface       string
	filter      string
	read        string
	promiscuous bool // 是否启用混杂模式
}

const (
	timestampFormt = "2006-01-02 15:04:05.000000"
)

func NewTcpTerm(ctx *ctx.Ctx) *TcpTerm {
	return &TcpTerm{
		ctx:    ctx,
		logger: ctx.Logger(),
	}
}

func (tcpTerm *TcpTerm) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&tcpTerm.iface, "interface", "i", "eth0", "interface to listen on")
	cmd.Flags().StringVarP(&tcpTerm.filter, "filter", "f", "tcp port 80", "package filter")
	cmd.Flags().StringVarP(&tcpTerm.read, "read", "r", "eth0.pcap", "set the local package")
	cmd.Flags().BoolVarP(&tcpTerm.promiscuous, "promiscuous", "p", false, "set promiscuous mode")
}

func (tcpTerm *TcpTerm) RunTcpTerm() *TcpTerm {
	packetSource, closer := tcpTerm.findSource()
	defer closer()
	view := tview.NewApplication()

	packetList := preparePacketList()
	packetDetail := preparePacketDetail()
	packetDump := preparePacketDump()

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(packetList, 0, 1, true).
		AddItem(packetDetail, 0, 1, false).
		AddItem(packetDump, 0, 1, false)
	frame := prepareFrame(layout)

	view.SetRoot(frame, true).SetFocus(packetList)

	app := &TcpTerm{
		src:        packetSource,
		view:       view,
		primitives: []tview.Primitive{packetList, packetDetail, packetDump},
		table:      packetList,
		detail:     packetDetail,
		dump:       packetDump,
		frame:      frame,
	}
	app.SwitchToTailMode()

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			app.Stop()
		}

		if event.Key() == tcell.KeyTAB {
			app.rotateView()
		}
		return event
	})

	packetList.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			app.SwitchToTailMode()
		}

		if key == tcell.KeyEnter {
			app.SwitchToSelectMode()
		}
	})

	packetList.SetSelectionChangedFunc(func(row int, column int) {
		app.displayDetailOf(row)
	})

	return app
}

var (
	snapshot_len int32         = 1024
	promiscuous  bool          = false
	timeout      time.Duration = 100 * time.Millisecond
)

func (tcpTerm *TcpTerm) findSource() (*gopacket.PacketSource, func()) {
	handler, err := tcpTerm.createHandle()
	if err != nil {
		tcpTerm.logger.Error("create handle error", zap.Error(err))
		return nil, nil
	}

	tcpTerm.handle = handler

	if tcpTerm.iface != "" {
		err := tcpTerm.handle.SetBPFFilter(tcpTerm.filter)
		if err != nil {
			log.Fatal(err)
		}
	}

	return gopacket.NewPacketSource(tcpTerm.handle, tcpTerm.handle.LinkType()), tcpTerm.handle.Close
}

func (tcpTerm *TcpTerm) findDevice() string {
	if tcpTerm.iface != "" {
		return tcpTerm.iface
	}
	devices, err := pcap.FindAllDevs()
	if err != nil {
		panic(err)
	}

	return devices[0].Name
}

func (tcpTerm *TcpTerm) createHandle() (*pcap.Handle, error) {
	if tcpTerm.read != "" {
		return pcap.OpenOffline(tcpTerm.read)
	} else {
		device := tcpTerm.findDevice()
		return pcap.OpenLive(device, snapshot_len, promiscuous, timeout)

	}
}

func (tcpTerm *TcpTerm) PacketListGenerator(refreshTrigger chan bool) {
	cnt := 0
	for {
		packet, err := tcpTerm.src.NextPacket()
		if err == io.EOF {
			return
		} else if err == nil {
			cnt++
			rowCount := tcpTerm.table.GetRowCount()

			tcpTerm.logger.Info("start count", zap.Int("count: ", cnt))

			tcpTerm.table.SetCell(rowCount, 0, tview.NewTableCell(strconv.Itoa(cnt)))
			tcpTerm.table.SetCell(rowCount, 1, tview.NewTableCell(packet.Metadata().Timestamp.Format(timestampFormt)))
			tcpTerm.table.SetCell(rowCount, 2, tview.NewTableCell(flowOf(packet)))
			tcpTerm.table.SetCell(rowCount, 3, tview.NewTableCell(strconv.Itoa(packet.Metadata().Length)))
			tcpTerm.table.SetCell(rowCount, 4, tview.NewTableCell(packet.Layers()[1].LayerType().String()))
			if len(packet.Layers()) > 2 {
				tcpTerm.table.SetCell(rowCount, 5, tview.NewTableCell(packet.Layers()[2].LayerType().String()))
			}

			tcpTerm.packets = append(tcpTerm.packets, packet)
			tcpTerm.logger.Info("end count", zap.Int("count: ", cnt))
		}
	}
}

func (app *TcpTerm) Ticker(refreshTrigger chan bool) {
	for {
		time.Sleep(100 * time.Millisecond)
		refreshTrigger <- true
	}
}

func (app *TcpTerm) Refresh(refreshTrigger chan bool) {
	for {
		_, ok := <-refreshTrigger
		if ok {
			app.view.Draw()
		}
	}
}

func (app *TcpTerm) Run() {
	refreshTrigger := make(chan bool)

	go app.PacketListGenerator(refreshTrigger)
	go app.Ticker(refreshTrigger)
	go app.Refresh(refreshTrigger)

	if err := app.view.Run(); err != nil {
		panic(err)
	}
}

func (app *TcpTerm) Stop() {
	app.view.Stop()
}

func (app *TcpTerm) SwitchToTailMode() {
	app.mode = TailMode
	// 先设置选择状态
	app.table.SetSelectable(false, false)
	// 然后滚动到底部
	app.table.ScrollToEnd()
	// 清除并重新添加文本
	app.frame.Clear().AddText("**Tail**", true, tview.AlignLeft, tcell.ColorGreen)

	app.frame.AddText("g: page top, G: page end, TAB: rotate panel, Enter: Detail mode", true, tview.AlignRight, tcell.ColorDefault)
}

func (app *TcpTerm) SwitchToSelectMode() {
	app.mode = SelectMode

	app.table.SetSelectable(true, false)
	row, _ := app.table.GetOffset()
	app.table.Select(row+1, 0)
	app.displayDetailOf(row + 1)

	app.frame.Clear().AddText("*Detail*", true, tview.AlignLeft, tcell.ColorBlue)
	app.frame.AddText("g: page top, G: page end, TAB: rotate panel, ESC: Tail mode", true, tview.AlignRight, tcell.ColorDefault)
}

func (app *TcpTerm) displayDetailOf(row int) {
	if row < 1 || row > len(app.packets) {
		return
	}

	app.detail.Clear().ScrollToBeginning()
	app.dump.Clear().ScrollToBeginning()

	packet := app.packets[row-1]

	fmt.Fprint(app.detail, packet.String())
	fmt.Fprint(app.dump, packet.Dump())
}

func (app *TcpTerm) rotateView() {
	idx, err := app.findPrimitiveIdx(app.view.GetFocus())
	if err != nil {
		panic(err)
	}

	nextIdx := idx + 1
	if nextIdx >= len(app.primitives) {
		nextIdx = 0
	}
	app.view.SetFocus(app.primitives[nextIdx])
}

func (app *TcpTerm) findPrimitiveIdx(p tview.Primitive) (int, error) {
	for i, primitive := range app.primitives {
		if p == primitive {
			return i, nil
		}
	}
	return 0, errors.New("Primitive not found")
}

func flowOf(packet gopacket.Packet) string {
	if packet.NetworkLayer() == nil {
		return "-"
	} else {
		return packet.NetworkLayer().NetworkFlow().String()
	}
}
