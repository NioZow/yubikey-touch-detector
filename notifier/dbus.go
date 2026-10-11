package notifier

import (
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	log "github.com/sirupsen/logrus"
)

const DBUS_IFACE string = "com.github.maximbaz.YubikeyTouchDetector"
const DBUS_PATH dbus.ObjectPath = "/com/github/maximbaz/YubikeyTouchDetector"

const PROP_GPG_STATE string = "GPGState"
const PROP_U2F_STATE string = "U2FState"
const PROP_HMAC_STATE string = "HMACState"
const PROP_LAST_EVENT string = "LastEvent"

var messagePropMap = map[Message]string{
	GPG_ON:   PROP_GPG_STATE,
	GPG_OFF:  PROP_GPG_STATE,
	U2F_ON:   PROP_U2F_STATE,
	U2F_OFF:  PROP_U2F_STATE,
	HMAC_ON:  PROP_HMAC_STATE,
	HMAC_OFF: PROP_HMAC_STATE,
}

var messageValueMap = map[Message]dbus.Variant{
	GPG_ON:   dbus.MakeVariant(uint32(1)),
	GPG_OFF:  dbus.MakeVariant(uint32(0)),
	U2F_ON:   dbus.MakeVariant(uint32(1)),
	U2F_OFF:  dbus.MakeVariant(uint32(0)),
	HMAC_ON:  dbus.MakeVariant(uint32(1)),
	HMAC_OFF: dbus.MakeVariant(uint32(0)),
}

type server struct{}

func SetupDbusNotifier(notifiers *sync.Map) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Error("Cannot establish dbus SessionBus connection ", err)
		return
	}
	defer conn.Close()

	reply, err := conn.RequestName(DBUS_IFACE,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Error("Cannot request dbus interface name ", err)
		return
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Error("dbus interface name already taken")
		return
	}

	propsSpec := map[string]map[string]*prop.Prop{
		DBUS_IFACE: {
			PROP_GPG_STATE: {
				Value:    uint32(0),
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					log.Debug(DBUS_IFACE, ".", c.Name, " changed to ", c.Value)
					return nil
				},
			},
			PROP_U2F_STATE: {
				Value:    uint32(0),
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					log.Debug(DBUS_IFACE, ".", c.Name, " changed to ", c.Value)
					return nil
				},
			},
			PROP_HMAC_STATE: {
				Value:    uint32(0),
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					log.Debug(DBUS_IFACE, ".", c.Name, " changed to ", c.Value)
					return nil
				},
			},
			PROP_LAST_EVENT: {
				Value:    "",
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					log.Debug(DBUS_IFACE, ".", c.Name, " changed to ", c.Value)
					return nil
				},
			},
		},
	}

	s := server{}
	err = conn.Export(s, DBUS_PATH, DBUS_IFACE)
	if err != nil {
		log.Error("dbus export server failed ", err)
		return
	}

	props, err := prop.Export(conn, DBUS_PATH, propsSpec)
	if err != nil {
		log.Error("dbus export propSpec failed ", err)
		return
	}
	n := &introspect.Node{
		Name: string(DBUS_PATH),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       DBUS_IFACE,
				Methods:    introspect.Methods(s),
				Properties: props.Introspection(DBUS_IFACE),
			},
		},
	}
	err = conn.Export(introspect.NewIntrospectable(n), DBUS_PATH, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		log.Error("dbus export introspect failed ", err)
		return
	}
	log.Debug("Connected to dbus session interface ", DBUS_IFACE)

	touch := make(chan Event, 10)
	notifiers.Store("notifier/dbus", touch)

	state := newTouchState()
	for {
		event := <-touch
		state.update(event.Message)
		if property, ok := messagePropMap[event.Message]; ok {
			if err := props.Set(DBUS_IFACE, property, messageValueMap[event.Message]); err != nil {
				log.Warn("dbus failed to update property ", property, ", ", err)
			}
		}
		if err := props.Set(DBUS_IFACE, PROP_LAST_EVENT, dbus.MakeVariant(event.JSONStringWithReasons(state.reasons()))); err != nil {
			log.Warn("dbus failed to update property ", PROP_LAST_EVENT, ", ", err)
		}
	}
}
