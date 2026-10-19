import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './styles.css'
import './workspace-theme.css'
import './glass-system.css'
import App from './App.vue'
import router from './router'
import { initializeTheme } from './theme'
import { initializeHomeAppearance } from './home-appearance'

initializeTheme()
initializeHomeAppearance()
createApp(App).use(router).use(ElementPlus).mount('#app')
