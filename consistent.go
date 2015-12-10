package ssf

import "sort"

const (
	NODE_ACTIVE  uint8 = 1
	NODE_LOADING uint8 = 2
	NODE_FAULT   uint8 = 3
)

type Node struct {
	Id          int32
	PartitionID int32
	Addr        string
	Status      uint8
}

func (node *Node) isActive() bool {
	return node.Status == NODE_ACTIVE
}

type Partition struct {
	Id    int32
	Addr  string
	Nodes []int32
}

//Consistent manager virutal nodes by consistent hash algorithm
type Consistent struct {
	NumberOfVirtualNode int
	partitions          map[string]*Partition
	partitionIDSeed     int32
}

func NewConsistent(numberOfVirtualNode int) *Consistent {
	c := new(Consistent)
	c.NumberOfVirtualNode = numberOfVirtualNode
	c.partitions = make(map[string]*Partition)
	return c
}

func NewConsistentCopy(other *Consistent) *Consistent {
	c := new(Consistent)
	c.NumberOfVirtualNode = other.NumberOfVirtualNode
	c.partitions = make(map[string]*Partition)
	c.partitionIDSeed = other.partitionIDSeed
	for _, part := range other.partitions {
		parition := new(Partition)
		*parition = *part
		c.partitions[part.Addr] = parition
	}
	return c
}

type partitionArray []*Partition

func (a partitionArray) Len() int           { return len(a) }
func (a partitionArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a partitionArray) Less(i, j int) bool { return a[i].Id < a[j].Id }

func (c *Consistent) update() {
	if len(c.partitions) == 0 {
		return
	}
	assignedNodes := make(map[int32]bool)
	numberOfVirtualNodePerPart := c.NumberOfVirtualNode / len(c.partitions)
	remainder := c.NumberOfVirtualNode % len(c.partitions)
	allPartitions := make([]*Partition, len(c.partitions))
	i := 0
	for _, partition := range c.partitions {
		allPartitions[i] = partition
		i++
	}
	sort.Sort(partitionArray(allPartitions))

	for _, partition := range allPartitions {
		if len(partition.Nodes) == 0 {
			continue
		}
		maxVirtualNodePerPart := numberOfVirtualNodePerPart
		if remainder > 0 {
			maxVirtualNodePerPart++
		}
		if len(partition.Nodes) > maxVirtualNodePerPart {
			partition.Nodes = partition.Nodes[0:maxVirtualNodePerPart]
		}
		if len(partition.Nodes) > numberOfVirtualNodePerPart {
			remainder--
		}
		for _, id := range partition.Nodes {
			assignedNodes[id] = true
		}
	}

	var unassingedNodes []int32
	for i := int32(0); i < int32(c.NumberOfVirtualNode); i++ {
		if _, ok := assignedNodes[i]; !ok {
			unassingedNodes = append(unassingedNodes, i)
		}
	}
	remainder = c.NumberOfVirtualNode % len(c.partitions)
	for _, partition := range allPartitions {
		if len(unassingedNodes) == 0 {
			break
		}
		maxVirtualNodePerPart := numberOfVirtualNodePerPart
		if remainder > 0 {
			maxVirtualNodePerPart++
		}
		if len(partition.Nodes) < maxVirtualNodePerPart {
			for len(partition.Nodes) < maxVirtualNodePerPart && len(unassingedNodes) > 0 {
				partition.Nodes = append(partition.Nodes, unassingedNodes[0])
				unassingedNodes = unassingedNodes[1:]
			}
		}
		if len(partition.Nodes) > numberOfVirtualNodePerPart {
			remainder--
		}
	}
}

func (c *Consistent) Add(server string) {
	if _, ok := c.partitions[server]; ok {
		return
	}
	part := &Partition{c.partitionIDSeed, server, nil}
	c.partitionIDSeed++
	c.partitions[server] = part
	c.update()
}

func (c *Consistent) Remove(server string) {
	if _, ok := c.partitions[server]; !ok {
		return
	}
	delete(c.partitions, server)
	c.update()
}

func (c *Consistent) Get(server string) []int32 {
	if part, ok := c.partitions[server]; ok {
		return part.Nodes
	}
	return nil
}

func (c *Consistent) Set(partition Partition) {
	var part Partition
	part = partition
	delete(c.partitions, part.Addr)
	c.partitions[part.Addr] = &part
	if partition.Id >= c.partitionIDSeed {
		c.partitionIDSeed = partition.Id + 1
	}
}

//Update update internal partitions
func (c *Consistent) Update() {
	c.update()
}

//Servers return sorted partitions
func (c *Consistent) Servers() []Partition {
	servers := make([]Partition, len(c.partitions))
	allPartitions := make(partitionArray, len(c.partitions))
	i := 0
	for _, partition := range c.partitions {
		allPartitions[i] = partition
		i++
	}
	sort.Sort(allPartitions)
	for i, partition := range allPartitions {
		servers[i] = *partition
	}
	return servers
}
