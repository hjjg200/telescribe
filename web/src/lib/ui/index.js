
import * as components from './components';
import './theme/index.scss';

// Font Awesome
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';

const UI = {
  install(Vue) {
    Vue.component('FontAwesome', FontAwesomeIcon);
    for(let key in components) {
      let component = components[key];
      Vue.component(component.name, component);
    }
  }
}

export default UI;