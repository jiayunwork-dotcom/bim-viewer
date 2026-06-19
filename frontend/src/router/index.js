import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/HomeView.vue')
  },
  {
    path: '/viewer/:modelId',
    name: 'Viewer',
    component: () => import('../views/ViewerView.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
