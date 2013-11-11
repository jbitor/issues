package gtkgui

import (
	"fmt"
	"github.com/jeremybanks/go-distributed/dht"
	"github.com/mattn/go-gtk/gtk"
	"math"
	"os"
	"time"
)

const targetNoteCount = 32

func init() {
	if os.Getenv("HOME") == "" {
		logger.Fatal("HOME not set\n")
		return
	}
}

func Main() {
	gtk.Init(nil)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("Distributed-GTK")

	dhtClientStatePath := os.Getenv("HOME") + "/.distributed-dht.benc"
	dhtClient, err := dht.OpenClient(dhtClientStatePath, false)

	_ = err // XXX

	_ = dhtClient

	notebook := gtk.NewNotebook()
	window.Add(notebook)

	notebook.AppendPage(MakeDhtVBox(dhtClient), gtk.NewLabel("DHT"))

	notebook.AppendPage(MakeTorrentVBox(), gtk.NewLabel("Torrent"))

	window.Connect("destroy", func() {
		dhtClient.Save()
		gtk.MainQuit()
	})

	window.SetSizeRequest(400, 500)
	window.SetResizable(false)

	window.ShowAll()

	gtk.Main()
}

func MakeDhtVBox(dhtClient dht.Client) *gtk.VBox {
	dhtPage := gtk.NewVBox(false, 0)

	statusLabel := gtk.NewLabel("Connection Status")
	statusLabel.SetAlignment(0.075, 1.0)
	dhtPage.Add(statusLabel)

	status := gtk.NewProgressBar()
	statusAlign := gtk.NewAlignment(0.45, 0.0, 0.90, 0.1)
	statusAlign.Add(status)
	dhtPage.Add(statusAlign)

	go func() {
		for {
			connectionInfo := dhtClient.ConnectionInfo()
			status.SetFraction(math.Min(float64(connectionInfo.GoodNodes)/targetNoteCount, 1.0))
			status.SetText(fmt.Sprintf(
				"%v Good Nodes\n(%v unknown, %v bad)",
				connectionInfo.GoodNodes,
				connectionInfo.UnknownNodes,
				connectionInfo.BadNodes,
			))
			time.Sleep(5 * time.Second)
		}
	}()

	return dhtPage
}

func MakeTorrentVBox() *gtk.VBox {
	dhtPage := gtk.NewVBox(false, 0)

	dhtPage.Add(gtk.NewLabel("NotImplemented :("))

	return dhtPage
}
