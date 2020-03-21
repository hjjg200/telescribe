
import vClickOutside from 'v-click-outside';

import * as components from './components';
import * as directives from './directives';
import './theme/index.scss';

// Font Awesome
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';

const UI = {
  install(Vue) {
    Vue.use(vClickOutside);
    Vue.component('FontAwesome', FontAwesomeIcon);
    
    [[components, 'component'], [directives, 'directive']].forEach(arr => {
      let [map, type] = arr;
      for(let key in map) {
        let item = map[key];
        Vue[type](item.name, item);
      }
    });
  }
}

export default UI;