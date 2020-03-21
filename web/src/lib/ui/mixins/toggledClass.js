
export default function(propName, className, initValue = false) {
  if(propName == undefined || propName == "") return;
  if(className == undefined || className == "") className = `is-${propName}`;
  return {
    props: {
      [propName]: {
        type: Boolean, default: initValue
      }
    },
    mounted() {
      this[`update${propName}`]();
    },
    watch: {
      [propName]: function() {
        this[`update${propName}`]();
      }
    },
    methods: {
      [`update${propName}`]: function() {
        this.$el.classList.toggle(className, this[propName]);
      }
    }
  }
}