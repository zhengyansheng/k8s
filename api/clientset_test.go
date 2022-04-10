package api

import (
	"fmt"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
)

var c = kubernetes.Clientset{}

func TestDeploymentList(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]string
		expect *kubernetes.Clientset
	}{
		{
			name: "list deployment",
			input: map[string]string{
				"kube_config": dmzClusterConfig,
				"namespace":   "cn-online",
			},
			expect: &c,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewClientSet(test.input["kube_config"])
			if err != nil {
				t.Fatal(err)
			}
			deployment, err := c.DeploymentListFormat(test.input["namespace"])
			if err != nil {
				t.Fatal(err)
			}

			//assert.EqualValues(t, test.expect, actual)
			fmt.Printf("%# v\n", pretty.Formatter(deployment))
		})
	}
}

func TestNodeList(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]string
		expect *kubernetes.Clientset
	}{
		{
			name: "list nodes",
			input: map[string]string{
				"kube_config": dmzClusterConfig,
			},
			expect: &c,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewClientSet(test.input["kube_config"])
			if err != nil {
				t.Fatal(err)
			}
			actual := c
			nodes, err := c.NodeListFormat()
			if err != nil {
				t.Fatal(err)
			}

			assert.EqualValues(t, test.expect, actual)
			fmt.Printf("%# v\n", pretty.Formatter(nodes))
		})
	}
}
