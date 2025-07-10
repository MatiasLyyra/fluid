#version 430

#define WORKGROUP_ITEMS {{ .WorkGroupItems }}

layout (local_size_x = {{ .WorkGroupSize }}) in;

uniform uint n_input;
uniform uint input_offset;
uniform uint sum_offset;

layout(std430, binding = 1) buffer input_data {
    uint input[];
};

shared uint cnt[WORKGROUP_ITEMS * 2];

{{ template "common_utilities" }}

void main()
{
    uint thread_id    = gl_LocalInvocationID.x;
    uint global_id    = gl_GlobalInvocationID.x;
    uint workgroup_id = gl_WorkGroupID.x;
    uint elem_id      = thread_id * 2;
    uint gelem_id     = global_id * 2;
    uint v1 = 0;
    uint v2 = 0;
    if (gelem_id     < n_input) v1 = input[input_offset + gelem_id    ];
    if (gelem_id + 1 < n_input) v2 = input[input_offset + gelem_id + 1];
    cnt[elem_id    ] = v1;
    cnt[elem_id + 1] = v2;
    uint sum;
    scan(thread_id, sum);
    if (thread_id == 0) input[sum_offset + workgroup_id] = sum;
    barrier();
    input[input_offset + gelem_id    ] = cnt[elem_id    ];
    input[input_offset + gelem_id + 1] = cnt[elem_id + 1];   
}