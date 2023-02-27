/*
Copyright 2023 Loggie.

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

package kubernetes

import (
	"context"
	"github.com/loggie-io/loggie/pkg/control"
	"github.com/loggie-io/loggie/pkg/core/interceptor"
	"github.com/loggie-io/loggie/pkg/core/log"
	"github.com/loggie-io/loggie/pkg/pipeline"
	"github.com/loggie-io/loggie/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/types"

	"github.com/loggie-io/loggie/pkg/core/cfg"
	"github.com/loggie-io/loggie/pkg/core/sink"
	"github.com/loggie-io/loggie/pkg/core/source"
	logconfigv1beta1 "github.com/loggie-io/loggie/pkg/discovery/kubernetes/apis/loggie/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func LogConfigToPipeline(lgc *logconfigv1beta1.LogConfig, client client.Client) (*control.PipelineConfig, error) {
	pipelineCfg := &control.PipelineConfig{}
	var pipRaws []pipeline.Config
	pip := lgc.Spec.Pipeline

	pipRaw := pipeline.Config{}
	pipRaw.Name = lgc.Name

	src, err := toPipelineSources(pip.Sources)
	if err != nil {
		return nil, err
	}
	pipRaw.Sources = src

	inter, err := toPipelineInterceptor(lgc.Spec.Pipeline.Interceptors, pip.InterceptorRef, client)
	if err != nil {
		return nil, err
	}
	pipRaw.Interceptors = inter

	sk, err := toPipelineSink(lgc.Spec.Pipeline.Sink, pip.SinkRef, client)
	if err != nil {
		return nil, err
	}
	pipRaw.Sink = sk

	pipRaws = append(pipRaws, pipRaw)

	pipelineCfg.Pipelines = pipRaws
	return pipelineCfg, nil
}

func LogConfigToPipelineStr(lgc *logconfigv1beta1.LogConfig, client client.Client) (string, error) {
	pipes, err := LogConfigToPipeline(lgc, client)
	if err != nil {
		return "", err
	}
	pipeData, err := yaml.Marshal(pipes)
	if err != nil {
		log.Error("marshal pipeline error: %v", err)
		return "", err
	}

	return string(pipeData), nil
}

func toPipelineSources(sources string) ([]*source.Config, error) {
	sourceCfg := make([]*source.Config, 0)
	err := cfg.UnPackFromRaw([]byte(sources), &sourceCfg).Do()
	if err != nil {
		return nil, err
	}
	return sourceCfg, nil
}

func toPipelineSink(sinkRaw string, sinkRef string, client client.Client) (*sink.Config, error) {
	// we use the sink in logConfig other than sinkRef if sink content is not empty
	var sinkStr string
	if sinkRaw != "" {
		sinkStr = sinkRaw
	} else {
		sk := logconfigv1beta1.Sink{}
		err := client.Get(context.Background(), types.NamespacedName{
			Name: sinkRef,
		}, &sk)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}

		sinkStr = sk.Spec.Sink
	}

	sinkConf := sink.Config{}
	err := cfg.UnPackFromRaw([]byte(sinkStr), &sinkConf).Do()
	if err != nil {
		return nil, err
	}

	return &sinkConf, nil
}

func toPipelineInterceptor(interceptorsRaw string, interceptorRef string, client client.Client) ([]*interceptor.Config, error) {
	var icp string
	if interceptorsRaw != "" {
		icp = interceptorsRaw
	} else {
		intercpt := logconfigv1beta1.Interceptor{}
		err := client.Get(context.Background(), types.NamespacedName{
			Name: interceptorRef,
		}, &intercpt)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}

		icp = intercpt.Spec.Interceptors
	}

	interConfList := make([]*interceptor.Config, 0)
	err := cfg.UnPackFromRaw([]byte(icp), &interConfList).Do()
	if err != nil {
		return nil, err
	}

	return interConfList, nil
}
