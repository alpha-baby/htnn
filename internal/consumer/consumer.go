// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumer

import (
	"encoding/json"
	"fmt"

	"mosn.io/htnn/internal/proto"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/filtermanager/model"
	"mosn.io/htnn/pkg/log"
	"mosn.io/htnn/pkg/plugins"
)

var (
	logger = log.DefaultLogger.WithName("consumer")
)

// Here we put Consumer in the internal/. So that we can access the Consumer's internal definition in other package
// of this repo, while hiding it from the plugin developers.

type Consumer struct {
	Auth    map[string]string              `json:"auth"`
	Filters map[string]*model.FilterConfig `json:"filters,omitempty"`

	// fields that set in the data plane
	namespace       string
	name            string
	generation      int
	ConsumerConfigs map[string]api.PluginConsumerConfig  `json:"-"`
	FilterConfigs   map[string]*model.ParsedFilterConfig `json:"-"`

	// fields that generated from the configuration
	CanSkipMethod  map[string]bool
	FilterNames    []string
	FilterWrappers []*model.FilterWrapper
}

func (c *Consumer) Marshal() string {
	// Consumer is defined to be marshalled to JSON, so err must be nil
	b, _ := json.Marshal(c)
	return string(b)
}

func (c *Consumer) Unmarshal(s string) error {
	return json.Unmarshal([]byte(s), c)
}

func (c *Consumer) InitConfigs() error {
	logger.Info("init configs for consumer", "name", c.name, "namespace", c.namespace)

	c.ConsumerConfigs = make(map[string]api.PluginConsumerConfig, len(c.Auth))
	for name, data := range c.Auth {
		p, ok := plugins.LoadHttpPlugin(name).(plugins.ConsumerPlugin)
		if !ok {
			return fmt.Errorf("plugin %s is not for consumer", name)
		}

		conf := p.ConsumerConfig()
		err := proto.UnmarshalJSON([]byte(data), conf)
		if err != nil {
			return fmt.Errorf("failed to unmarshal consumer config for plugin %s: %w", name, err)
		}

		err = conf.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate consumer config for plugin %s: %w", name, err)
		}

		c.ConsumerConfigs[name] = conf
	}

	c.FilterConfigs = make(map[string]*model.ParsedFilterConfig, len(c.Filters))
	for name, data := range c.Filters {
		p := plugins.LoadHttpFilterFactoryAndParser(name)
		if p == nil {
			return fmt.Errorf("plugin %s not found", name)
		}

		conf, err := p.ConfigParser.Parse(data.Config, nil)
		if err != nil {
			return fmt.Errorf("%w during parsing plugin %s in consumer", err, name)
		}

		c.FilterConfigs[name] = &model.ParsedFilterConfig{
			Name:         name,
			ParsedConfig: conf,
			Factory:      p.Factory,
		}
	}

	return nil
}

// Implement pkg.filtermanager.api.Consumer
func (c *Consumer) Name() string {
	return c.name
}

func (c *Consumer) PluginConfig(name string) api.PluginConsumerConfig {
	return c.ConsumerConfigs[name]
}
