/*
Copyright 2021.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	operatorv1alpha1 "github.com/example/nginx-operator/api/v1alpha1"
	"github.com/example/nginx-operator/assets"
	"github.com/nats-io/nats.go"
	"github.com/samuel/go-zookeeper/zk"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NginxOperatorReconciler reconciles a NginxOperator object
type NginxOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var operatorID string

// zooker conections and node
var zkConn *zk.Conn
var operatorPath string

// Se o operador é lider ou não
var isLeader bool

// nats conection
var nc *nats.Conn

// estrura para armazenar a informação de cada operador
type OperatorInfo struct {
	CostPredict    float64
	AllocationCost float64
	Deployed       bool
	Limit          float64
}

// Custo previsto dos operadores
var costs = make(map[string]OperatorInfo)

// Custo alocações dos operadores
//var allocations = make(map[string]float64)

//+kubebuilder:rbac:groups=operator.example.com,resources=nginxoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.example.com,resources=nginxoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.example.com,resources=nginxoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NginxOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *NginxOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	//metrics.ReconcilesTotal.Inc()

	logger := log.FromContext(ctx)

	//Cr atual do operador
	operatorCR := &operatorv1alpha1.NginxOperator{}

	//get operador existente
	err := r.Get(ctx, req.NamespacedName, operatorCR)

	if err != nil && errors.IsNotFound(err) {
		logger.Info("Operator resource object not found.")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Error getting operator resource object")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "OperatorDegraded",
			Status:             metav1.ConditionTrue,
			Reason:             operatorv1alpha1.ReasonCRNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()), Message: fmt.Sprintf("ERRO: unable to get operator custom resource: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	fmt.Println("Operator ID Reconcile 77777: ")
	fmt.Println(operatorID)

	// Get the list of all ephemeral nodes under the same path
	operators, _, err2 := zkConn.Children("/operators")
	if err2 != nil {
		return ctrl.Result{}, err2
		//panic(err2)
	}

	/*isLeader = true

	for _, operator := range operators {
		fmt.Println(operator)
		if operator < strings.TrimPrefix(operatorPath, "/operators/") {
			fmt.Println("ENTREI NO FALSE")
			isLeader = false
			break
		}
	}
	if isLeader {
		fmt.Println("Este operador é lider")
	} else {
		fmt.Println("Este operador não é lider")
	}*/

	sort.Strings(operators)
	leaderOperator := operators[0]
	fmt.Println("Operador lider " + leaderOperator)

	//Atualizar map com os custos (eliminar operadores que já não se encontram ativos)
	aux := make(map[string]OperatorInfo)

	if len(costs) > 0 {

		for k, v := range costs {
			for _, operator := range operators {
				if k == operator {
					aux[k] = v
					break
				}
			}
		}
		costs = aux
	}

	//Print do map
	for k, v := range costs {
		fmt.Printf("ID: %s\n", k)
		fmt.Printf("CostPredict: %.6f\n", v.CostPredict)
		fmt.Printf("AllocationCost: %.6f\n", v.AllocationCost)
		fmt.Printf("Deployed: %v\n", v.Deployed)
		fmt.Println("--------------------")
	}

	fmt.Println("Pedido Allocation")
	jsonAllocation := AtualCostRequest(req.Namespace)
	allocationCost := decodeJsonAllocationApiKubecost(jsonAllocation, req.Namespace)
	fmt.Println("alocation cost:")
	fmt.Println(allocationCost)

	/*fmt.Println("======Limite de Custo====")
	fmt.Println(*operatorCR.Spec.AppLimitCost)

	fmt.Println("=======APP=======")
	fmt.Println(operatorCR.Spec.IsDeployed)
	message := "ola sou o operador " + operatorID

	if err := nc.Publish("test", []byte(message)); err != nil {
		panic(err)
	}
	fmt.Println("Messagem enviada")*/

	jsonSTR := PredictCostRequest(operatorCR.Spec.Replicas, req.Namespace)

	totalCost := decodeJsonPreditApiKubecost(jsonSTR)

	messageCost := "ID: " + operatorID + " CostPredict: " +
		strconv.FormatFloat(totalCost, 'f', 6, 64) +
		" AllocationCost: " + strconv.FormatFloat(allocationCost, 'f', 6, 64) +
		" Deployed: " + strconv.FormatBool(operatorCR.Spec.IsDeployed) +
		" Limit: " + strconv.Itoa(int(*operatorCR.Spec.AppLimitCost))

	if err := nc.Publish("PredictCost", []byte(messageCost)); err != nil {
		panic(err)
	}
	fmt.Println("Messagem Custo previsto enviada")

	//Print do map
	/*for k, v := range costs {
		fmt.Printf("ID: %s\n", k)
		fmt.Printf("CostPredict: %.6f\n", v.CostPredict)
		fmt.Printf("AllocationCost: %.6f\n", v.AllocationCost)
		fmt.Printf("Deployed: %v\n", v.Deployed)
		fmt.Println("--------------------")
	}*/

	// Subscreva para ouvir mensagens de custo previsto
	nc.QueueSubscribe("PredictCost", "operators."+operatorID, func(m *nats.Msg) {

		fmt.Printf("Received message: %s\n", string(m.Data))

		split := strings.Split(string(m.Data), " ")
		if len(split) < 9 {
			fmt.Println("Formato errado")
		}
		if len(split) == 10 {

			id := split[1]
			cost, err := strconv.ParseFloat(split[3], 64)
			if err != nil {
				panic(err)
			}
			allocation, err := strconv.ParseFloat(split[5], 64)
			if err != nil {
				panic(err)
			}

			deployed, err := strconv.ParseBool(split[7])
			if err != nil {
				panic(err)
			}

			appLimit, err := strconv.ParseFloat(split[9], 64)
			if err != nil {
				panic(err)
			}

			/*fmt.Println(id)
			fmt.Println(cost)
			fmt.Println(allocation)
			fmt.Println(deployed)
			fmt.Println(appLimit)
			fmt.Println("=======================================================\n===============================\n===========")
			*/
			info := OperatorInfo{
				CostPredict:    cost,
				AllocationCost: allocation,
				Deployed:       deployed,
				Limit:          appLimit,
			}

			costs[id] = info
		}
	})

	if allocationCost < float64(*operatorCR.Spec.AppLimitCost) {
		fmt.Println("O preco da aplicacao encontra-se dentro do limite!")
	} else if allocationCost >= float64(*operatorCR.Spec.AppLimitCost) && operatorCR.Spec.IsDeployed == true {
		fmt.Println("O preço da aplicacao nao se encontra dentro do limite!")
		fmt.Println("Necessário tomar decisão!")

		messageCost := "ID: " + operatorID + " Reason: " + "LimiteViolated"

		if err := nc.Publish("Decisons", []byte(messageCost)); err != nil {
			panic(err)
		}

		fmt.Println("Mensagem Publicada para Decisons")
	}

	if operatorID == leaderOperator {
		fmt.Println("Sou o lider, vou observar se existem decisões a tomar")
		// Subscreva para ouvir mensagens de custo previsto
		nc.QueueSubscribe("Decisons", "operators."+operatorID, func(m *nats.Msg) {

			fmt.Printf("Received message: %s\n", string(m.Data))
			fmt.Println("TOMAR DECISÃO!!!!")
			split := strings.Split(string(m.Data), " ")
			if len(split) < 4 {
				fmt.Println("Formato errado")
			}

			id := split[1]
			//reason := split[3]
			/*if err != nil {
				fmt.Println("Panico")
				panic(err)
			}*/

			/*.Println(id)
			fmt.Println(reason)
			fmt.Println("---------------------------------")
			*/

			//Fazer uma queue com as mensagens para evitar que se tomam duas decisões ao mesmo
			//tempo, e depois os valores não estejam atualizados

			selected := migrateApp(id)
			fmt.Println("SELECIONADO: " + selected)

			if selected != "" {
				messageOrder := "LeaderID: " + operatorID + " Destination: " + selected + " AppDeploy: true"

				if err := nc.Publish("Orders", []byte(messageOrder)); err != nil {
					panic(err)
				}
				fmt.Println("Messagem enviada orders!")

				messageOrder2 := "LeaderID: " + operatorID + " Destination: " + id + " AppDeploy: false"

				if err := nc.Publish("Orders", []byte(messageOrder2)); err != nil {
					panic(err)
				}
				fmt.Println("Messagem enviada orders! 2")

			}

		})
	}

	nc.QueueSubscribe("Orders", "operators."+operatorID, func(m *nats.Msg) {

		fmt.Printf("Received Order: %s\n", string(m.Data))
		split := strings.Split(string(m.Data), " ")

		if len(split) < 6 {
			fmt.Println("Formato errado")
		}

		//id_leader := split[1]
		id_selected := split[3]
		order, err := strconv.ParseBool(split[5])
		if err != nil {
			panic(err)
		}

		/*fmt.Println(id_leader)
		fmt.Println(id_selected)
		fmt.Println(order)
		fmt.Println("---------------------------------")
		*/

		//Verificar se sou o operador responsável por executar a ordem e se quem fez a ordem foi o lider
		if operatorID == id_selected /*&& leaderOperator == id_leader*/ {
			//executar a ordem
			if order == true {
				fmt.Println("Entrei no true")
				fmt.Println(operatorCR.Spec.IsDeployed)

				//Cr atual do operador
				operatorCR := &operatorv1alpha1.NginxOperator{}

				//get operador existente
				err := r.Get(ctx, req.NamespacedName, operatorCR)
				if err != nil {
					logger.Info("Failed to get Operator resource.", "error", err)
					// Handle the error, return or retry as needed.
					//return ctrl.Result{}, err
				}

				//Update Cr to deploy app
				operatorCR.Spec.IsDeployed = true

				err = r.Update(ctx, operatorCR)
				//err = r.Update(ctx, operatorCR)
				if err != nil && errors.IsNotFound(err) {
					logger.Info("Operator resource object not found.")
					//return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
					operatorCR.Spec.IsDeployed = false
				} else if err != nil {
					logger.Info("Operator resource couldnt be updated.")
					operatorCR.Spec.IsDeployed = false
					fmt.Println(err)
					fmt.Println(err)
					fmt.Println(err)
					fmt.Println(err)
					fmt.Println("ola")
					//return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
				}
				fmt.Println("Entrei no true apos update")
				fmt.Println(operatorCR.Spec.IsDeployed)

			} else {
				//Cr atual do operador
				fmt.Println("Entrei no else")
				fmt.Println(operatorCR.Spec.IsDeployed)
				operatorCR := &operatorv1alpha1.NginxOperator{}

				//get operador existente
				err := r.Get(ctx, req.NamespacedName, operatorCR)

				//Tirar o deployment logo aqui ?
				operatorCR.Spec.IsDeployed = false
				err = r.Update(ctx, operatorCR)
				//err = r.Update(ctx, operatorCR)
				if err != nil && errors.IsNotFound(err) {
					logger.Info("Operator resource object not found.")
					operatorCR.Spec.IsDeployed = true
					//return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
				} else if err != nil {
					logger.Info("Operator resource couldnt be updated 1.")
					fmt.Println(err)
					operatorCR.Spec.IsDeployed = true

					//return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
				}
				fmt.Println("Entrei no else apos update")
				fmt.Println(operatorCR.Spec.IsDeployed)
			}
		}
	})

	if operatorCR.Spec.IsDeployed {
		fmt.Println("A Dar deploy ou verificar deployment")
		deployment := &appsv1.Deployment{}

		create := false

		err = r.Get(ctx, req.NamespacedName, deployment)
		if err != nil && errors.IsNotFound(err) {
			create = true
			//Buscar deployment do nginx ao ficheiro nginx_deployment.yaml
			deployment = assets.GetDeploymentFromFile("manifests/nginx_deployment.yaml")
		} else if err != nil {
			logger.Error(err, "Error getting existing Nginx deployment.")
			meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
				Type:               "OperatorDegraded",
				Status:             metav1.ConditionTrue,
				Reason:             operatorv1alpha1.ReasonDeploymentNotAvailable,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Message:            fmt.Sprintf("unable to get operand deployment: %s", err.Error()),
			})
			return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
		}

		deployment.Namespace = req.Namespace
		deployment.Name = req.Name

		if operatorCR.Spec.Replicas != nil {
			deployment.Spec.Replicas = operatorCR.Spec.Replicas
		}
		if operatorCR.Spec.Port != nil {
			deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = *operatorCR.Spec.Port
		}
		ctrl.SetControllerReference(operatorCR, deployment, r.Scheme)

		if create {
			err = r.Create(ctx, deployment)
		} else {
			err = r.Update(ctx, deployment)
		}
		if err != nil {
			meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
				Type:               "OperatorDegraded",
				Status:             metav1.ConditionTrue,
				Reason:             operatorv1alpha1.ReasonOperandDeploymentFailed,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Message:            fmt.Sprintf("ERRO: unable to update operand deployment: %s NAMESPACE %s", err.Error(), deployment.Namespace),
			})
			return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
		}

		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "OperatorDegraded",
			Status:             metav1.ConditionFalse,
			Reason:             operatorv1alpha1.ReasonSucceeded,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            "operator successfully reconciling",
		})
		r.Status().Update(ctx, operatorCR)

		/*condition, err := conditions.InClusterFactory{Client: r.Client}.
		NewCondition(apiv2.ConditionType(apiv2.Upgradeable))

		if err != nil {
			return ctrl.Result{}, err
		}*/

		/*err = condition.Set(ctx, metav1.ConditionTrue,
		conditions.WithReason("OperatorUpgradeable"),
		conditions.WithMessage("The operator is currently upgradeable"))

		if err != nil {
			return ctrl.Result{}, err
		}*/

		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})

	} else if operatorCR.Spec.IsDeployed == false {
		fmt.Println("Não há deployment para fazer")
		deployment := &appsv1.Deployment{}
		err = r.Get(ctx, req.NamespacedName, deployment)
		if err == nil {
			err = r.Delete(ctx, deployment)
			if err != nil {
				// Lidar com o erro ao excluir o deployment
				return ctrl.Result{}, err
			}
		} else {
			fmt.Println("Não existe deployment para apagar")
		}

	}

	return ctrl.Result{}, nil
}

/*
func generateOperatorID() {

	rand.Seed(time.Now().UnixNano())

	operatorID = rand.New(rand.NewSource(time.Now().UnixNano())).Int63()

	fmt.Println("Operator ID: ")
	fmt.Println(operatorID)
}*/

func connectToNats() error {
	var err error
	nc, err = nats.Connect("nats://my-nats.nats")
	if err != nil {
		fmt.Println("Could not conect to nats server")
		return err
	}
	return nil
}

func connectToZookeeper() error {
	var err error
	zkConn, _, err = zk.Connect([]string{"ab707b55c178048478d36379cc26f4f7-1423547521.eu-west-1.elb.amazonaws.com:2181"}, time.Second*5)
	if err != nil {
		fmt.Println("Could not conect to zookeper server")
		return err
	}
	return nil

}

func createEphemeralNode() error {
	var err error
	operatorPath, err = zkConn.Create("/operators/operator-", []byte{}, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		return err
	}
	fmt.Println("Before trim")
	fmt.Println(operatorPath)
	operatorID = strings.TrimPrefix(operatorPath, "/operators/")
	fmt.Println("TRIM")
	fmt.Println(operatorID)
	return nil

}

func migrateApp(id string) string {

	var selectedOperator string

	lowest := math.MaxFloat64

	fmt.Println("Função de migrar")

	for k, v := range costs {
		fmt.Println("Opearador idddddd: " + k)
		fmt.Println(v.AllocationCost)
		fmt.Println(v.CostPredict)
		fmt.Println(v.Deployed)
		fmt.Println(v.Limit)
		if (v.AllocationCost+v.CostPredict <= v.Limit) && v.Deployed == false {
			fmt.Println("entrei no if")
			if v.CostPredict < lowest {
				lowest = v.CostPredict
				selectedOperator = k
			}
		}
	}

	if selectedOperator == "" {
		fmt.Println("No Operators Available")
		return ""
	} else {
		//fmt.Println(idMaisBaixo)
		fmt.Println("A migrar de " + id + " para o operador " + selectedOperator)
		return selectedOperator
	}

}

// SetupWithManager sets up the controller with the Manager.
func (r *NginxOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//generateOperatorID()
	if err := connectToNats(); err != nil {
		panic(err)
	}
	//defer nc.Close()
	fmt.Println("Connected to NATS server!")
	if err := connectToZookeeper(); err != nil {
		panic(err)
	}
	fmt.Println("Connected to Zookeeper server!")
	if err := createEphemeralNode(); err != nil {
		panic(err)
	}
	fmt.Println("Node ephemeral created")
	fmt.Println(operatorPath)

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.NginxOperator{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
