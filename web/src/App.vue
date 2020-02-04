<template>
  <div id="app">
    <main>
      <Sidebar>
        <SidebarLabel>Hosts</SidebarLabel>
        <SidebarItem>
          <strong>Title</strong>
          Details
        </SidebarItem>
        <SidebarItem>
          <strong>Abcd</strong>
          Details
        </SidebarItem>
      </Sidebar>
      <section>
        <Button>ABC</Button>
        <Dropdown name="fruit">
          <DropdownItem value="apple">Apple</DropdownItem>
          <DropdownItem value="banana">Banana</DropdownItem>
        </Dropdown>
        <Checkbox name="okay" value="false">Not OK</Checkbox>
      </section>
    </main>
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
<style lang="scss" src="@/style.scss"></style>