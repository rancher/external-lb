package model

import (
)


type LBRecord struct {
	Vip		 string	
	ServiceName string
	Nodes	[]LBNode
}

type LBNode struct {
	HostIP	 string
	Port	 string
}