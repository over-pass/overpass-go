package commandamqp

import "github.com/over-pass/overpass-go/src/overpass"

func logInvokerInvalidMessageID(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID string,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker ignored AMQP message, '%s' is not a valid message ID",
		peerID.ShortString(),
		msgID,
	)
}

func logInvokerIgnoredMessage(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	err error,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker ignored AMQP message %s, %s",
		peerID.ShortString(),
		msgID.ShortString(),
		err,
	)
}

func logUnicastCallBegin(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	target overpass.PeerID,
	ns string,
	cmd string,
	traceID string,
	payload *overpass.Payload,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker began unicast '%s::%s' call %s to %s [%s] >>> %s",
		peerID.ShortString(),
		ns,
		cmd,
		msgID.ShortString(),
		target.ShortString(),
		traceID,
		payload,
	)
}

func logBalancedCallBegin(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	ns string,
	cmd string,
	traceID string,
	payload *overpass.Payload,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker began '%s::%s' call %s [%s] >>> %s",
		peerID.ShortString(),
		ns,
		cmd,
		msgID.ShortString(),
		traceID,
		payload,
	)
}

func logCallEnd(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	ns string,
	cmd string,
	traceID string,
	payload *overpass.Payload,
	err error,
) {
	if !logger.IsDebug() {
		return
	}

	switch e := err.(type) {
	case nil:
		logger.Log(
			"%s invoker completed '%s::%s' call %s successfully [%s] <<< %s",
			peerID.ShortString(),
			ns,
			cmd,
			msgID.ShortString(),
			traceID,
			payload,
		)
	case overpass.Failure:
		var message string
		if e.Message != "" {
			message = ": " + e.Message
		}

		logger.Log(
			"%s invoker completed '%s::%s' call %s with '%s' failure%s [%s] <<< %s",
			peerID.ShortString(),
			ns,
			cmd,
			msgID.ShortString(),
			e.Type,
			message,
			traceID,
			payload,
		)
	default:
		logger.Log(
			"%s invoker completed '%s::%s' call %s with error [%s] <<< %s",
			peerID.ShortString(),
			ns,
			cmd,
			msgID.ShortString(),
			traceID,
			err,
		)
	}
}

func logAsyncRequest(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	ns string,
	cmd string,
	traceID string,
	payload *overpass.Payload,
	err error,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker sent asynchronous '%s::%s' call request %s [%s] >>> %s",
		peerID.ShortString(),
		ns,
		cmd,
		msgID.ShortString(),
		traceID,
		payload,
	)
}

func logAsyncResponse(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	ns string,
	cmd string,
	traceID string,
	payload *overpass.Payload,
	err error,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker received asynchronous '%s::%s' call response %s [%s] >>> %s",
		peerID.ShortString(),
		ns,
		cmd,
		msgID.ShortString(),
		traceID,
		payload,
	)
}

func logBalancedExecute(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	ns string,
	cmd string,
	traceID string,
	payload *overpass.Payload,
	err error,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker sent '%s::%s' execution %s [%s] >>> %s",
		peerID.ShortString(),
		ns,
		cmd,
		msgID.ShortString(),
		traceID,
		payload,
	)
}

func logMulticastExecute(
	logger overpass.Logger,
	peerID overpass.PeerID,
	msgID overpass.MessageID,
	ns string,
	cmd string,
	traceID string,
	payload *overpass.Payload,
	err error,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker sent multicast '%s::%s' execution %s [%s] >>> %s",
		peerID.ShortString(),
		ns,
		cmd,
		msgID.ShortString(),
		traceID,
		payload,
	)
}

func logInvokerStart(
	logger overpass.Logger,
	peerID overpass.PeerID,
	preFetch int,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker started (pre-fetch: %d)",
		peerID.ShortString(),
		preFetch,
	)
}

func logInvokerStopping(
	logger overpass.Logger,
	peerID overpass.PeerID,
	pending int,
) {
	if !logger.IsDebug() {
		return
	}

	logger.Log(
		"%s invoker stopping gracefully (pending: %d)",
		peerID.ShortString(),
		pending,
	)
}

func logInvokerStop(
	logger overpass.Logger,
	peerID overpass.PeerID,
	err error,
) {
	if !logger.IsDebug() {
		return
	}

	if err == nil {
		logger.Log(
			"%s invoker stopped",
			peerID.ShortString(),
		)
	} else {
		logger.Log(
			"%s invoker stopped: %s",
			peerID.ShortString(),
			err,
		)
	}
}
