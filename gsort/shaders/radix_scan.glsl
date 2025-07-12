#version 430

#define WORKGROUP_ITEMS {{ .WorkGroupItems }}

layout (local_size_x = {{ .WorkGroupSize }}) in;

uniform uint n_input;
uniform uint offset;
uniform uint n_workgroups;

{{ template "input_type" . }}

layout(std430, binding = 1) buffer input_data_buffer {
    InputData input[];
};

layout(std430, binding = 2) buffer output_data_buffer {
    uint local_prefix_sum[];
};

layout(std430, binding = 3) buffer sums_data_buffer {
    uint block_sums[];
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
    uint base_id      = workgroup_id * WORKGROUP_ITEMS;

    // Initialize v1 and v2 to values outside of range [0,3] to prevent counting them as 0, 1, 2 or 3,
    // in case input data size is not aligned to WORKGROUP_ITEMS.
    uint v1 = 4;
    uint v2 = 4;
    if (gelem_id     < n_input) v1 = ((input[gelem_id    ].key >> offset) & 0x3u);
    if (gelem_id + 1 < n_input) v2 = ((input[gelem_id + 1].key >> offset) & 0x3u);

    uvec4 bit_sum1 = uvec4(0u);
    uvec4 bit_sum2 = uvec4(0u);
    for (uint b = 0; b < 4; b++)
    {
        cnt[elem_id    ] = v1 == b ? 1 : 0;
        cnt[elem_id + 1] = v2 == b ? 1 : 0;
        uint block_sum;
        scan(thread_id, block_sum);
        if (thread_id == 0) {
            uint idx = b * n_workgroups + workgroup_id;
            block_sums[idx] = block_sum;
        }
        barrier();
        bit_sum1[b] = cnt[elem_id    ];
        bit_sum2[b] = cnt[elem_id + 1];
    }

    if (gelem_id < n_input) {
        local_prefix_sum[gelem_id] = bit_sum1[v1];
    } else {
        local_prefix_sum[gelem_id] = 0;
    }
    if (gelem_id + 1 < n_input) {
        local_prefix_sum[gelem_id + 1] = bit_sum2[v2];
    } else {
        local_prefix_sum[gelem_id + 1] = 0;
    }
}