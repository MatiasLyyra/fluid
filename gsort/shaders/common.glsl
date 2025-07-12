{{ define "common_utilities" }}
void scan(uint thread_id, out uint block_sum)
{
    uint elem_id = thread_id * 2;
    uint offset = 1u;
    for (uint d = WORKGROUP_ITEMS >> 1; d > 0; d >>= 1)
    {
        barrier();
        if (thread_id < d)
        {
            uint ai = offset * (elem_id + 1) - 1;
            uint bi = offset * (elem_id + 2) - 1;

            cnt[bi] += cnt[ai];
        }
        offset <<= 1;
    }

    if (thread_id == 0)
    {
        block_sum = cnt[WORKGROUP_ITEMS - 1]; 
        cnt[WORKGROUP_ITEMS - 1] = 0;
    }

    for (uint d = 1; d < WORKGROUP_ITEMS; d <<= 1)
    {
        offset >>= 1;
        barrier();

        if (thread_id < d)
        {
            uint ai = offset * (elem_id + 1) - 1;
            uint bi = offset * (elem_id + 2) - 1;
            uint t = cnt[ai];
            cnt[ai] = cnt[bi];
            cnt[bi] += t;
        }
    }
}
{{ end }}

{{ define "input_type" }}
{{- if and (eq .PaddingBefore 0) (eq .PaddingAfter 0) }}
struct InputData {
    uint key;
};
{{- else if eq .PaddingBefore 0 }}
struct InputData {
    uint key;
    uint _padding2[{{ .PaddingAfter }}];
};
{{- else if eq .PaddingAfter 0}}
struct InputData {
    uint _padding1[{{ .PaddingBefore }}];
    uint key;
};
{{- else }}
struct InputData {
    uint _padding1[{{ .PaddingBefore }}];
    uint key;
    uint _padding2[{{ .PaddingAfter }}];
};
{{- end }}
{{ end }}