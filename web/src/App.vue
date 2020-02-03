<template>
  <div id="app">
    <Button>ABC</Button>
    <Dropdown name="fruit">
      <DropdownItem value="apple">Apple</DropdownItem>
      <DropdownItem value="banana">Banana</DropdownItem>
    </Dropdown>
  </div>
</template>

<script>
export default {
  name: "App",
  data() {
    return {
    };
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Open+Sans:400,700&display=swap);
</style>