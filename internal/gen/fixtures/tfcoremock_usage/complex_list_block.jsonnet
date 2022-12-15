local tfcoremock = import './tfcoremock/main.libsonnet';
local tf = import 'github.com/tf-libsonnet/core/main.libsonnet';

tfcoremock.complex_resource.new('foo', list_block=[
  tfcoremock.complex_resource.list_block.new(string='hello'),
])
+ tfcoremock.complex_resource.withListBlockMixin(
  'foo',
  tfcoremock.complex_resource.list_block.new(string='world'),
)
+ tf.withOutput(
  'foo',
  '${join(" ", [for blk in tfcoremock_complex_resource.foo.list_block : blk.string])}',
)
