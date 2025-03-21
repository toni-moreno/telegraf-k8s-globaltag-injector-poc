# POC Añadir globaltags dinamicamente de labels del nodo el que se ejecuta telegraf.
Esta poc pretende mostrar una posible solución escalable para que con una sola configuración de daemonset de Telegraf de forma dinámica se puedan importar información de labels de nodo tal como lo hace node_exporter.


##  Necesidad

Si tenemos extraer info de SO con telegraf de un nodo de EKS que dispone de ciertos labels por ejemplo: 

```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    alpha.kubernetes.io/provided-node-ip: X.Y.Z.T
    node.alpha.kubernetes.io/ttl: "0"
    volumes.kubernetes.io/controller-managed-attach-detach: "true"
  creationTimestamp: "2025-10-17T10:40:12Z"
  labels:
    alpha.eksctl.io/nodegroup-name: eks-spot-nodes
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: m6i.2xlarge
    beta.kubernetes.io/os: linux
    eks.amazonaws.com/capacityType: ON_DEMAND
    eks.amazonaws.com/nodegroup: MyNodeGroup
    eks.amazonaws.com/nodegroup-image: ami-0XXXXXXXXX
    failure-domain.beta.kubernetes.io/region: eu-west-1
    failure-domain.beta.kubernetes.io/zone: eu-west-1a
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: ip-X-Y-Z-T.eu-west-1.compute.internal
    kubernetes.io/os: linux
    node.kubernetes.io/instance-type: m6i.2xlarge
    topology.ebs.csi.aws.com/zone: eu-west-1a
    topology.kubernetes.io/region: eu-west-1
    topology.kubernetes.io/zone: eu-west-1a
    workergroup: mygroup
```

Para ello podemos  aprovechar una característica de telegraf y es que permite añadir configuraciones en el fichero `/etc/telegraf/telegraf.conf` con variables de entorno, por ejemplo:

```toml
[global_tags]

  tipo_nodo = "${NODE_KUBERNETES_IO_INSTANCE_TYPE}"
  region = "${TOPOLOGY_KUBERNETES_IO_REGION}"
  zone = "${TOPOLOGY_KUBERNETES_IO_ZONE}"
  workergroup = "${WORKERGROUP}"

[[inputs.mem]]

[[outputs.influxdb]]
  urls = ["https://myinfluxdb.xxxxx.com"]
  skip_database_creation = true
  password = "${INFLUX_PASSWORD}"

```

##  Problema detectado

Necesitamos añadir en global_tags configuraciones que solo aparecen definidas como labels en el nodo donde se ejecuta.

Aunque es factible setear variables de entorno dinámicas  mediante en uso de `valueFrom`, `fieldRef`, `fieldPath` como se ve en el ejemplo, se observa 

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dapi-envars-fieldref
spec:
  containers:
    - name: my-telegraf
      image: docker.io/library/telegraf:latest
      env:
        - name: MY_NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: MY_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: MY_POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: MY_POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: MY_POD_SERVICE_ACCOUNT
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
```
Sin embargo vemos que solo hay opción de setear:
* [valores definidos en el pod](https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#use-pod-fields-as-values-for-environment-variables)
* [valores definidos en el contenedor](https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#use-container-fields-as-values-for-environment-variables)

No existe opción de pasarle información referente al **nodo** en el que se ejecuta el pod.

## Solución propuesta.

Puesto que necesitamos extraer información del nodo, de algun modo vamos a tener que consultar esa información y pasarsela como variables de entorno 

Para ello solo tenemos que hacer 2 cosas.

1.- ejecutar un proceso antes de que se levante el telegraf que genere en un fichero todos esos valores en cada nodo en formato.

Esto se ejecuta con añadiendo un init container que ejecuta un proceso que hace una consutal via api de k8s equivalente a `kubectl get node <NODENAME>` y genera un fichero segun indica la variable INIT_ENV_FILE `/shared/init.env` ( NOTA: shared es un volumen persistente para que telegraf lo encuentre después)

```yaml
      initContainers:
      - name: init-helper
        image: ghcr.io/toni-moreno/k8s-node-label-extractor:latest
        imagePullPolicy: IfNotPresent
        env:
        - name: NODENAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: INIT_ENV_FILE
          value: /shared/init.env
```

2.- Cargar las variables de entorno antes de ejecutar telegraf.
Eso se consigue añadiendo un comando custom a la ejecución en lugar del que hay por defecto.

```yaml
      containers:
      - name: telegraf
        image: docker.io/library/telegraf:latest
        imagePullPolicy: IfNotPresent
        command: ['/bin/bash', '-c', 'source /shared/init.env && /usr/bin/telegraf'] 
```

Con este sistema basado en initContainer no hará falta crear una imagen a medida para cada version de telegraf y se podra usar la imagen del fabricante y actualizar tantas veces como queramos sin necesidad de recrear imagenes.

