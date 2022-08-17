package task

func Run() {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
		channel_4 = make(chan bool, 1)
	)
	go task_self_judgment(channel_1)
	go task_RemoveInvalidFiles(channel_2)
	go task_HandlingChallenges(channel_3)
	go task_SpaceManagement(channel_4)
	for {
		select {
		case <-channel_1:
			go task_self_judgment(channel_1)
		case <-channel_2:
			go task_RemoveInvalidFiles(channel_2)
		case <-channel_3:
			go task_HandlingChallenges(channel_3)
		case <-channel_4:
			go task_SpaceManagement(channel_4)
		}

	}
}
