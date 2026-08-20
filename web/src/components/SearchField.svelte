<script lang="ts">
  let { value = $bindable(''), label = 'Search' }: { value?: string; label?: string } = $props()
</script>
<label class="relative block w-full sm:max-w-xs">
  <span class="sr-only">{label}</span>
  <input
    type="search"
    {value}
    oninput={(event) => value = event.currentTarget.value}
    aria-label={label}
    placeholder={label}
    class="field-input"
  />
</label>
