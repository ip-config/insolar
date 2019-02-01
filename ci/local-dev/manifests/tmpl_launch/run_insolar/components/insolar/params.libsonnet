{
  global: {
    // User-defined global parameters; accessible to all component and environments, Ex:
    // replicas: 4,
  },
  components: {
    "insolar":{ 
    	num_heavies: 1,
    	num_lights: 5,
    	num_virtuals: 4,
    	hostname: "seed",
    	domain: "bootstrap",
    	tcp_transport_port: 7900,
    	},
    	"utils":{
    		get_num_nodes : $.components.insolar.num_heavies + $.components.insolar.num_lights + $.components.insolar.num_virtuals,
            host_template : $.components.insolar.hostname + "-%d." + $.components.insolar.domain + ":" + $.components.insolar.tcp_transport_port
    	}
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
  },
}
