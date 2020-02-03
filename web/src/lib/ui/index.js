import * as components from './components';
import './theme/index.scss';

const UI = {
  install(Vue) {
    for(let key in components) {
      let component = components[key];
      Vue.component(component.name, component);
    }
  }
}

export default UI;