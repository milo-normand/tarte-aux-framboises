package legohat

import (
	"context"
	"fmt"
	"log"
	"time"
)

type deviceDriver struct {
	devices []*deviceRegistration
	adaptor *Adaptor
}

func (drv *deviceDriver) waitForConnect() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, d := range drv.devices {
		registration := drv.adaptor.awaitMessage(d.id, ConnectedMessage)

		d.toDevice <- []byte(fmt.Sprintf("port %d ; select ; echo 0\r", d.id))
		d.toDevice <- []byte(fmt.Sprintf("list\r"))

		log.Printf("Waiting for %s to connect on port %d...\n", Motor, d.id)

		_, err := d.waitForEventOnDevice(ctx, ConnectedMessage, registration.conduit)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deviceRegistration) waitForEventOnDevice(ctx context.Context, awaitedMsgType DeviceMessageType, conduit <-chan DeviceEvent) (rawData []byte, err error) {
	select {
	case e := <-conduit:
		log.Printf("Received message on port %d: %v\n", d.id, e)
		switch e.msgType {
		case awaitedMsgType:
			log.Printf("Got awaited message %s on %s device at port %d", awaitedMsgType, d.class, d.id)
			return e.data, nil
		case TimeoutMessage:
			log.Printf("Got awaited message %s on %s device at port %d", timeoutMessage, d.class, d.id)
			return e.data, fmt.Errorf("received timeout from %s device on port %d", d.class, d.id)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out waiting for message %s for device %s on port %d", awaitedMsgType, d.class, d.id)
	}

	return nil, fmt.Errorf("unreachable code reached")
}
