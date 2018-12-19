package v1beta1

import (
	"context"
	"sync"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"github.com/rancher/norman/objectclient/dynamic"
	"github.com/rancher/norman/restwatch"
	"k8s.io/client-go/rest"
)

type (
	contextKeyType        struct{}
	contextClientsKeyType struct{}
)

type Interface interface {
	RESTClient() rest.Interface
	controller.Starter

	PodDisruptionBudgetsGetter
}

type Clients struct {
	Interface Interface

	PodDisruptionBudget PodDisruptionBudgetClient
}

type Client struct {
	sync.Mutex
	restClient rest.Interface
	starters   []controller.Starter

	podDisruptionBudgetControllers map[string]PodDisruptionBudgetController
}

func Factory(ctx context.Context, config rest.Config) (context.Context, controller.Starter, error) {
	c, err := NewForConfig(config)
	if err != nil {
		return ctx, nil, err
	}

	cs := NewClientsFromInterface(c)

	ctx = context.WithValue(ctx, contextKeyType{}, c)
	ctx = context.WithValue(ctx, contextClientsKeyType{}, cs)
	return ctx, c, nil
}

func ClientsFrom(ctx context.Context) *Clients {
	return ctx.Value(contextClientsKeyType{}).(*Clients)
}

func From(ctx context.Context) Interface {
	return ctx.Value(contextKeyType{}).(Interface)
}

func NewClients(config rest.Config) (*Clients, error) {
	iface, err := NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return NewClientsFromInterface(iface), nil
}

func NewClientsFromInterface(iface Interface) *Clients {
	return &Clients{
		Interface: iface,

		PodDisruptionBudget: &podDisruptionBudgetClient2{
			iface: iface.PodDisruptionBudgets(""),
		},
	}
}

func NewForConfig(config rest.Config) (Interface, error) {
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = dynamic.NegotiatedSerializer
	}

	restClient, err := restwatch.UnversionedRESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &Client{
		restClient: restClient,

		podDisruptionBudgetControllers: map[string]PodDisruptionBudgetController{},
	}, nil
}

func (c *Client) RESTClient() rest.Interface {
	return c.restClient
}

func (c *Client) Sync(ctx context.Context) error {
	return controller.Sync(ctx, c.starters...)
}

func (c *Client) Start(ctx context.Context, threadiness int) error {
	return controller.Start(ctx, threadiness, c.starters...)
}

type PodDisruptionBudgetsGetter interface {
	PodDisruptionBudgets(namespace string) PodDisruptionBudgetInterface
}

func (c *Client) PodDisruptionBudgets(namespace string) PodDisruptionBudgetInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &PodDisruptionBudgetResource, PodDisruptionBudgetGroupVersionKind, podDisruptionBudgetFactory{})
	return &podDisruptionBudgetClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}
