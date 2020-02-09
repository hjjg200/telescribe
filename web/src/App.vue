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
        
        <Client v-for="(client, fullName) in abstract.clientMap"
          :key="fullName"
          :body="client"></Client>
        <!--
        <Button>ABC</Button>
        <Dropdown name="fruit" style="width: 300px">
          <DropdownItem value="apple">Apple</DropdownItem>
          <DropdownItem value="banana">Banana</DropdownItem>
        </Dropdown>
        <Checkbox color="#56e" name="okay" value="false" v-model="temp1">Not OK</Checkbox>
        -->
      </section>
    </main>
  </div>
</template>

<script>
import Client from '@/components/Client.vue';
export default {
  name: "App",
  components: {Client},
  computed: {
    abstract() {
      return this.$root.abstract;
    },
    options() {
      return this.$root.options;
    }
  },
  data() {
    return {
      temp1: true
    };
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Open+Sans:400,700&display=swap);
</style>
<style lang="scss" src="@/style.scss"></style>