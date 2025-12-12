/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.

SPDX-License-Identifier: MPL-2.0

File: org-chart.go
Description: Organization chart generator
Author: tengzl33t
*/

package main

import (
	"context"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"gopkg.in/yaml.v3"
	"maps"
	"os"
	"slices"
)

type Person struct {
	Title    string `yaml:"title"`
	Position string `yaml:"position"`
}

type Group struct {
	Title     string   `yaml:"title"`
	Positions []string `yaml:"positions"`
}

type Relation struct {
	Manager string   `yaml:"manager"`
	Report  []string `yaml:"report"`
}

type Config struct {
	People    map[string]Person `yaml:"people"`
	Groups    map[string]Group  `yaml:"groups"`
	Relations []Relation        `yaml:"relations"`
}

type NodeInfo struct {
	Node        *cgraph.Node
	IsCluster   bool
	ClusterName string
	RepNode     *cgraph.Node
}

func createEdges(cfg Config, nodes map[string]*NodeInfo, graph *cgraph.Graph) {
	for _, relation := range cfg.Relations {
		if len(relation.Report) == 0 {
			println("Warning: Relation of " + relation.Manager + " has no report entries")
			continue
		}
		fromInfo := nodes[relation.Manager]
		for _, report := range relation.Report {
			toInfo := nodes[report]

			if fromInfo == nil || toInfo == nil {
				println("Warning: Missing node for relation " + relation.Manager + "-> " + report)
				continue
			}

			if fromInfo.RepNode == nil || toInfo.RepNode == nil {
				println("Warning: Missing representative node for relation " + relation.Manager + "-> " + report)
				continue
			}

			edge, err := graph.CreateEdgeByName("", fromInfo.RepNode, toInfo.RepNode)
			if err != nil {
				println("Error creating edge " + err.Error())
				continue
			}

			if fromInfo.IsCluster {
				edge.SetLogicalTail(fromInfo.ClusterName)
			}
			if toInfo.IsCluster {
				edge.SetLogicalHead(toInfo.ClusterName)
			}

			edge.SetColor("#666666")
			edge.SetMinLen(2)
		}
	}
}

func createPerson(id string, person Person, graph *cgraph.Graph, nodes map[string]*NodeInfo) *cgraph.Node {
	item, _ := graph.CreateNodeByName(id)
	label := person.Title + "\\n" + person.Position
	item.SetLabel(label)
	item.SetShape("box")
	item.SetStyle("rounded,filled")
	item.SetFillColor("#E8F0FE")
	nodes[id] = &NodeInfo{
		Node:      item,
		IsCluster: false,
		RepNode:   item,
	}
	return item
}

func createGroups(cfg Config, nodes map[string]*NodeInfo, graph *cgraph.Graph) {
	for _, id := range slices.Sorted(maps.Keys(cfg.Groups)) {
		group := cfg.Groups[id]
		if len(group.Positions) == 0 {
			println("Warning: Group '" + id + "' has no positions")
			continue
		}

		clusterName := "cluster_" + id
		cluster, _ := graph.CreateSubGraphByName(clusterName)
		cluster.SetLabel(group.Title)
		cluster.SetStyle("rounded,filled")
		cluster.SetBackgroundColor("#AAAAAA")

		var posNodes []*cgraph.Node

		for _, position := range group.Positions {
			posNodes = append(posNodes, createPerson(position, cfg.People[position], cluster, nodes))
		}

		var repNode *cgraph.Node
		if len(posNodes) > 0 {
			middleIndex := len(posNodes) / 2
			repNode = posNodes[middleIndex]
		}

		nodes[id] = &NodeInfo{
			Node:        nil,
			IsCluster:   true,
			ClusterName: clusterName,
			RepNode:     repNode,
		}
	}
}

func getGroupPositions(cfg Config) []string {
	var positions []string
	for _, v := range cfg.Groups {
		positions = append(positions, v.Positions...)
	}
	return positions
}

func createPeople(cfg Config, nodes map[string]*NodeInfo, graph *cgraph.Graph) {
	gropedPositions := getGroupPositions(cfg)

	for _, id := range slices.Sorted(maps.Keys(cfg.People)) {
		if !slices.Contains(gropedPositions, id) {
			createPerson(id, cfg.People[id], graph, nodes)
		}
	}
}

func renderGraph(
	gv *graphviz.Graphviz,
	cfg Config,
) {
	ctx := context.Background()
	graph, err := gv.Graph()
	if err != nil {
		panic(err)
	}

	defer func(graph *graphviz.Graph) {
		err := graph.Close()
		if err != nil {
			panic(err)
		}
	}(graph)

	graph.SetRankDir("TB")
	graph.SetCompound(true)

	nodes := map[string]*NodeInfo{}

	createPeople(cfg, nodes, graph)
	createGroups(cfg, nodes, graph)
	createEdges(cfg, nodes, graph)

	file, err := os.OpenFile("file.svg", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	if err := gv.Render(ctx, graph, graphviz.SVG, file); err != nil {
		panic(err)
	}
	println("Graph rendered successfully to file.svg")
}

func main() {
	data, err := os.ReadFile("org.yaml")
	if err != nil {
		panic(err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}

	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		panic(err)
	}

	defer func(g *graphviz.Graphviz) {
		err := g.Close()
		if err != nil {
			panic(err)
		}
	}(gv)

	renderGraph(gv, cfg)
}
