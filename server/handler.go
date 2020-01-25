package server

func HandleConnection(conn *ConBroker) {
	conn._wg.Add(2)
	go func() {
		defer func() {
			conn._wg.Done()
			conn.Close()
		}()

		for {
			//接收消息

		}

	}()

	go func() {
		defer conn._wg.Done()
		for {
			select {
			case <-conn._closed:
				goto Exit
			case <-conn._kicker.C:
				conn.Kicker()
			case msg := <-conn._queue:
				if msg != nil {

				}
			}
		}
	Exit:
	}()
}
