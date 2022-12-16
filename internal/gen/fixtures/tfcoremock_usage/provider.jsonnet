local tfcoremock = import './tfcoremock/main.libsonnet';
local tf = import 'github.com/tf-libsonnet/core/main.libsonnet';

tfcoremock.provider.new(use_only_state=true, alias='foo', src='hashicorp/tfcoremock', version='=0.1.2')
+ tfcoremock.provider.new(alias='bar')
