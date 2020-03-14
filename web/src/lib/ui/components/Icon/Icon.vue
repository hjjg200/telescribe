<template>
  <span class="ui-icon">
    <span v-if="type === 'error'">
      <svg ref="svg" width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M16 8C16 12.4183 12.4183 16 8 16C3.58172 16 0 12.4183 0 8C0 3.58172 3.58172 0 8 0C12.4183 0 16 3.58172 16 8Z" fill="#FD0F0F"/>
        <path d="M9.14062 10.3984H7.00781L6.67188 2.625H9.47656L9.14062 10.3984ZM6.59375 12.8203C6.59375 12.4349 6.73177 12.1198 7.00781 11.875C7.28906 11.625 7.63802 11.5 8.05469 11.5C8.47135 11.5 8.81771 11.625 9.09375 11.875C9.375 12.1198 9.51562 12.4349 9.51562 12.8203C9.51562 13.2057 9.375 13.5234 9.09375 13.7734C8.81771 14.0182 8.47135 14.1406 8.05469 14.1406C7.63802 14.1406 7.28906 14.0182 7.00781 13.7734C6.73177 13.5234 6.59375 13.2057 6.59375 12.8203Z" fill="white"/>
      </svg>
    </span>
    <span v-else-if="type === 'warning'">
      <svg ref="svg" width="19" height="17" viewBox="0 0 19 17" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M7.76795 1C8.53775 -0.333333 10.4623 -0.333334 11.2321 1L18.5933 13.75C19.3631 15.0833 18.4008 16.75 16.8612 16.75H2.13879C0.599187 16.75 -0.363064 15.0833 0.406736 13.75L7.76795 1Z" fill="#FFB800"/>
        <path d="M10.6406 11.3984H8.50781L8.17188 3.625H10.9766L10.6406 11.3984ZM8.09375 13.8203C8.09375 13.4349 8.23177 13.1198 8.50781 12.875C8.78906 12.625 9.13802 12.5 9.55469 12.5C9.97135 12.5 10.3177 12.625 10.5938 12.875C10.875 13.1198 11.0156 13.4349 11.0156 13.8203C11.0156 14.2057 10.875 14.5234 10.5938 14.7734C10.3177 15.0182 9.97135 15.1406 9.55469 15.1406C9.13802 15.1406 8.78906 15.0182 8.50781 14.7734C8.23177 14.5234 8.09375 14.2057 8.09375 13.8203Z" fill="black"/>
      </svg>
    </span>
    <span v-else-if="type === 'green-light'">
      <svg ref="svg" width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="8" cy="8" r="8" fill="#00C472"/>
      </svg>
    </span>
  </span>
</template>

<script>
export default {
  name: "Icon",
  props: {
    type: String
  },
  mounted() {
    let rem = parseFloat(getComputedStyle(document.documentElement).fontSize);
    let svg = this.$refs.svg;
    if(svg) {
      let w = svg.width.baseVal.value / rem;
      let h = svg.height.baseVal.value / rem;
      svg.style.width  = `${w}em`;
      svg.style.height = `${h}em`;
    } else {
      this.$el.style.display = "none";
    }
  }
}
</script>