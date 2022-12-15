local tfcoremock = import './tfcoremock/main.libsonnet';
local tf = import 'github.com/tf-libsonnet/core/main.libsonnet';

tfcoremock.complex_resource.new('foo', list=[{ string: 'test-render-library' }])
+ tfcoremock.complex_resource.withListMixin('foo', [{ string: 'another' }, { string: 'one' }])
+ tf.withOutput(
  'foo',
  '${join(" ", [for foo in tfcoremock_complex_resource.foo.list : foo.string])}',
)
