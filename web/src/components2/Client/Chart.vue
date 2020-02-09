<template>
  <div class="chart"></div>
</template>

<script>

// Static functions
function remToPx(rem) {
  return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
}

function keysEqual(a, b) {
  if(a === b) return true;
  if(a == null || b == null) return false;
  if(a.length != b.length) return false;
  a = a.slice(0).sort();
  b = b.slice(0).sort();
  for (var i = 0; i < a.length; ++i) {
      if(a[i] !== b[i]) return false;
  }
  return true;
}

// Exported component
export default {
  name: "Chart",
  data() {
    return {};
  },
  computed: {
    client: function() {
      return this.$parent;
    },
    options: function() {
      return this.client.app.options
    },
    csvBox: function() {
      return this.client.csvBox;
    }
  },
  created: async function() {

  },
  mounted: async function() {
  },
  methods: {
    // Private

    _fetch: async function() {
  
    },

    _draw: async function() {

    },

    _scale: function() {

    },

    // Public

    update: async function() {
 
    },

    keys: function(arr) {
      if(arr === undefined) return this._keys;
      this._keys = arr;
      this.update();
      return this;
    },

    duration(d) {
      if(d === undefined) return this._duration;
      this._duration = d;
      this._draw();
      return this;
    }

  }
}
</script>

<style lang="scss" scoped>

</style>