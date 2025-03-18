package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func isRunningInKubernetes() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount")
	return !os.IsNotExist(err)
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
func main() {
	// Verificar si el binario se está ejecutando dentro de un contenedor de Kubernetes

	var config *rest.Config
	err := error(nil)
	if isRunningInKubernetes() {
		log.Info().Msg("This binary is running inside a Kubernetes container")
		// Configuración dentro del pod
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else {
		log.Info().Msg("This binary is not running inside a Kubernetes container")
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

		// Construir la configuración desde el archivo kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Error().Err(err).Msg("Error building kubeconfig")
			os.Exit(1)
		}
	}

	// Ruta al archivo de configuración de kubeconfig

	// Crear cliente de Kubernetes
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Kubernetes client")
		os.Exit(1)
	}

	// Obtener el nombre del nodo desde la variable de entorno
	nodeName := os.Getenv("NODENAME")
	if nodeName == "" {
		log.Error().Msg("Error getting NODENAME environment variable")
		os.Exit(1)
	}

	// Obtener información del nodo
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Error getting node information")
		os.Exit(1)
	}

	// Extraer y mostrar las etiquetas del nodo
	labels := node.ObjectMeta.Labels
	//formatea las etiquetas del nodo KEY=VALUE,
	// donde KEY es el nombre de la key del label en mayusculas substituyendo los puntos por _ y los / por - y
	// value es el valor del valor de la label.
	formattedLabels := make(map[string]string)
	for key, value := range labels {
		replacer := strings.NewReplacer("=", "_", ".", "_", "-", "_", "/", "_")
		formattedKey := strings.ToUpper(replacer.Replace(key))
		formattedLabels[formattedKey] = value
		log.Info().Msgf("created variable: %s=%s", formattedKey, value)
	}
	// Obtener el nombre del archivo desde la variable de entorno
	envFileName := os.Getenv("INIT_ENV_FILE")
	if envFileName == "" {
		log.Error().Msg("Error getting INIT_ENV_FILE environment variable")
		os.Exit(1)
	}

	// Crear el archivo
	file, err := os.Create(envFileName)
	if err != nil {
		log.Error().Err(err).Msg("Error creating environment file")
		os.Exit(1)
	}
	defer file.Close()

	// Escribir las etiquetas formateadas en el archivo
	for key, value := range formattedLabels {
		_, err := file.WriteString(fmt.Sprintf("export %s=\"%s\"\n", key, value))
		if err != nil {
			log.Error().Err(err).Msg("Error writing to environment file")
			os.Exit(1)
		}
	}

	log.Info().Msgf("Environment file %s created successfully", envFileName)
}
