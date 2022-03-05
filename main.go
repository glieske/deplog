package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/urfave/cli/v2"
)

func main() {
	var follow bool
	var container string
	var count int

	app := &cli.App{
		Name:    "deplog",
		Version: "v0.1.1",
		Authors: []*cli.Author{
			{
				Name:  "Paweł Cyman",
				Email: "pawel@cyman.xyz",
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "follow",
				Aliases:     []string{"f"},
				Value:       false,
				Usage:       "follow the logs",
				Destination: &follow,
			},
			&cli.StringFlag{
				Name:        "container",
				Aliases:     []string{"c"},
				Usage:       "specify which container",
				Required:    true,
				Destination: &container,
			},
			&cli.IntFlag{
				Name:        "count",
				Aliases:     []string{"n"},
				Usage:       "how many logs per pod to query",
				Destination: &count,
			},
		},
		Usage: "dep your logs",
		Action: func(c *cli.Context) error {
			countSet := c.IsSet("count")
			deployment := c.Args().Get(0)
			if deployment == "" {
				log.Fatal("Provide a deployment")
			}
			getLogs(deployment, container, follow, int64(count), countSet)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getLogs(
	deployment string,
	container string,
	follow bool,
	count int64,
	countSet bool,
) {
	client := getClientSet()
	currentNamespace := getCurrentNamespace()
	pods := getPodList(client, currentNamespace)

	re, err := regexp.Compile(`^` + deployment + `\-[0-9a-f]+\-[0-9a-z]+`)
	if err != nil {
		fmt.Println(err)
	}

	wg := new(sync.WaitGroup)

	for _, pod := range pods.Items {
		if !re.Match([]byte(pod.GetName())) {
			continue
		}
		wg.Add(1)
		go getPodLogs(wg, *client, currentNamespace, pod.GetName(), container, follow, count, countSet)
	}

	wg.Wait()
}

func getPodLogs(
	wg *sync.WaitGroup,
	clientSet kubernetes.Clientset,
	namespace string,
	podName string,
	containerName string,
	follow bool,
	count int64,
	countSet bool,
) {
	podLogOptions := v1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
	}
	if countSet {
		podLogOptions.TailLines = &count
	}
	podLogRequest := clientSet.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		fmt.Println(err)
	}
	defer stream.Close()

	colorReset := "\033[0m"
	colorBlue := "\033[34m"
	logBuf := ""

	for {
		buf := make([]byte, 2000)
		numBytes, err := stream.Read(buf)

		if err == io.EOF {
			break
		}
		if numBytes == 0 {
			continue
		}
		if err != nil {
			fmt.Println(err)
		}

		message := string(buf[:numBytes])
		for _, c := range message {
			if c == '\n' {
				fmt.Println(colorBlue + podName + " | " + colorReset + logBuf)
				logBuf = ""
			} else {
				logBuf += string(c)
			}
		}
	}

	wg.Done()
}

func getCurrentNamespace() string {
	clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		panic(err)
	}
	currentContext := clientCfg.CurrentContext
	currentNamespace := clientCfg.Contexts[currentContext].Namespace
	return currentNamespace
}

func getClientSet() *kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		panic(err)
	}
	clientSet, _ := kubernetes.NewForConfig(config)

	return clientSet
}

func getPodList(client *kubernetes.Clientset, currentNamespace string) *v1.PodList {
	pods, err := client.CoreV1().Pods(currentNamespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err)
	}

	return pods
}
