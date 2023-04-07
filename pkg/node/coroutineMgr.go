package node

func (node *Node) CoroutineMgr() {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
	)
	go node.task_self_judgment(channel_1)
	go node.task_RemoveInvalidFiles(channel_2)
	go node.task_HandlingChallenges(channel_3)

	for {
		select {
		case <-channel_1:
			go node.task_self_judgment(channel_1)
		case <-channel_2:
			go node.task_RemoveInvalidFiles(channel_2)
		case <-channel_3:
			go node.task_HandlingChallenges(channel_3)
		}
	}
}
