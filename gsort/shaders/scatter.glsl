#version 430

#define WORKGROUP_ITEMS {{ .WorkGroupItems }}

layout (local_size_x = {{ .WorkGroupSize }}) in;

uniform uint n_input;
uniform uint n_workgroups;
uniform uint offset;

layout(std430, binding = 1) buffer input_buffer {
    uint input[];
};

layout(std430, binding = 2) buffer output_buffer {
    uint output_data[];
};

layout(std430, binding = 3) buffer local_prefix_sum_buffer {
    uint local_prefix_sum[];
};

layout(std430, binding = 4) buffer block_sums_buffer {
    uint block_sums[];
};

void main()
{
    uint thread_id    = gl_LocalInvocationID.x;
    uint global_id    = gl_GlobalInvocationID.x;
    uint workgroup_id = gl_WorkGroupID.x;
    uint elem_id      = thread_id * 2;
    uint gelem_id     = global_id * 2;

    uint v1 = 0u;
    uint v2 = 0u;

    if (gelem_id < n_input) {
        v1 = input[gelem_id];
    }
    if (gelem_id + 1 < n_input) {
        v2 = input[gelem_id + 1];
    }

    uint b1 = (v1 >> offset) & 0x3u;
    uint b2 = (v2 >> offset) & 0x3u;

    uint pos1 = local_prefix_sum[gelem_id    ];
    uint pos2 = local_prefix_sum[gelem_id + 1];

    uint idx1 = b1 * n_workgroups + workgroup_id;
    uint idx2 = b2 * n_workgroups + workgroup_id;

    uint block1 = block_sums[idx1];
    uint block2 = block_sums[idx2];

    pos1 += block1;
    pos2 += block2;

    // if (gelem_id     < n_input) output_data[gelem_id    ] = pos1;
    // if (gelem_id + 1 < n_input) output_data[gelem_id + 1] = pos2;
    if (gelem_id     < n_input && pos1 < n_input) output_data[pos1] = v1;
    if (gelem_id + 1 < n_input && pos2 < n_input) output_data[pos2] = v2;

}