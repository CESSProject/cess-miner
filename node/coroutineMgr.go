package node

func (n *Node) CoroutineMgr() {
	var (
		ch_spaceMgr = make(chan bool, 1)
		ch_fileMgr  = make(chan bool, 1)
		// channel_3   = make(chan bool, 1)
	)
	go n.spaceMgr(ch_spaceMgr)
	go n.fileMgr(ch_fileMgr)
	// go n.task_self_judgment(channel_1)
	// go n.task_RemoveInvalidFiles(channel_2)
	// go n.task_HandlingChallenges(channel_3)

	for {
		select {
		case <-ch_spaceMgr:
			go n.spaceMgr(ch_spaceMgr)
		case <-ch_fileMgr:
			go n.fileMgr(ch_fileMgr)
			// case <-channel_2:
			// 	go n.task_RemoveInvalidFiles(channel_2)
			// case <-channel_3:
			// 	go n.task_HandlingChallenges(channel_3)
		}
	}
}
