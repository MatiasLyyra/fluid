#version 430

#define WORKGROUP_ITEMS {{ .WorkGroupItems }}

layout (local_size_x = {{ .WorkGroupSize }}) in;

uniform uint input_offset;
uniform uint sum_offset;

layout(std430, binding = 1) buffer input_data {
    uint input[];
};

void main()
{
    uint thread_id    = gl_LocalInvocationID.x;
    uint global_id    = gl_GlobalInvocationID.x;
    uint workgroup_id = gl_WorkGroupID.x;
    uint elem_id      = thread_id * 2;
    uint gelem_id     = global_id * 2;

    input[input_offset + gelem_id    ] += input[sum_offset + workgroup_id];
    input[input_offset + gelem_id + 1] += input[sum_offset + workgroup_id];
}