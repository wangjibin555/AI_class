// pages/course/list/list.js
const courseAPI = require('../../../apis/course.js')
const authAPI = require('../../../apis/auth.js')
const auth = require('../../../utils/auth.js')
const { debounce } = require('../../../utils/debounce.js')

Page({
  data: {
    courseList: [],
    isLoading: false,
    isLoadingMore: false,
    hasMore: true,
    searchKeyword: '',
    
    // 课件类型切换
    courseType: 'mine', // 'mine' | 'public'
    courseTypeOptions: [
      { value: 'mine', label: '我的课件' },
      { value: 'public', label: '公开课件' }
    ],
    
    // 搜索相关
    searchHistory: [],
    showSearchHistory: false,
    searchDebounced: null,
    
    // 筛选选项
    filters: {
      status: 'all',
      category: 'all',
      sort: 'created_at',
      order: 'desc'
    },
    
    // 分页参数
    pagination: {
      page: 1,
      limit: 50,
      total: 0
    },
    
    // 状态选项
    statusOptions: [
      { value: 'all', label: '全部' },
      { value: 'generating', label: '生成中' },
      { value: 'completed', label: '已完成' },
      { value: 'failed', label: '失败' }
    ],
    
    // 分类选项
    categoryOptions: [
      { value: 'all', label: '全部分类' },
      { value: 'general', label: '通用' },
      { value: 'programming', label: '编程' },
      { value: 'business', label: '商业' },
      { value: 'education', label: '教育' },
      { value: 'science', label: '科学' },
      { value: 'literature', label: '文学' },
      { value: 'history', label: '历史' },
      { value: 'art', label: '艺术' }
    ],
    
    // 排序选项
    sortOptions: [
      { value: 'created_at', label: '创建时间' },
      { value: 'updated_at', label: '更新时间' },
      { value: 'view_count', label: '观看次数' },
      { value: 'like_count', label: '点赞次数' }
    ],
    
    // 界面状态
    showFilters: false,
    isRefreshing: false,
    
    // 用户信息
    userInfo: null
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    console.log('课件列表页加载成功')
    
    // 检查登录状态
    if (!auth.isLoggedIn()) {
      auth.requireLogin()
      return
    }

    // 初始化防抖搜索（暂时禁用自动搜索）
    this.setData({
      searchDebounced: debounce(() => {
        // 暂时禁用自动搜索，只显示提示
        wx.showToast({
          title: '搜索功能暂未开放',
          icon: 'none',
          duration: 2000
        })
      }, 500)
    })

    this.loadUserInfo()
    this.loadSearchHistory()
    this.loadFilterState()
    this.loadCourseList()
  },

  // 导航栏事件处理
  goBack() {
    wx.navigateBack({
      delta: 1
    })
  },

  showMoreOptions() {
    wx.showActionSheet({
      itemList: ['导出课程', '分享', '设置', '帮助'],
      success: (res) => {
        switch(res.tapIndex) {
          case 0:
            wx.showToast({ title: '导出功能开发中', icon: 'none' })
            break
          case 1:
            wx.showToast({ title: '分享功能开发中', icon: 'none' })
            break
          case 2:
            wx.navigateTo({ url: '/pages/settings/settings' })
            break
          case 3:
            wx.navigateTo({ url: '/pages/help/help' })
            break
        }
      }
    })
  },

  refreshList() {
    this.refreshCourseList()
  },

  /**
   * 生命周期函数--监听页面显示
   */
  onShow() {
    console.log('课件列表页显示')
    
    // 检查登录状态
    if (!auth.isLoggedIn()) {
      auth.requireLogin()
      return
    }

    // 刷新用户信息和课程列表
    this.loadUserInfo()
    this.refreshCourseList()
  },

  /**
   * 页面相关事件处理函数--监听用户下拉动作
   */
  onPullDownRefresh() {
    console.log('用户下拉刷新')
    this.refreshCourseList()
  },

  /**
   * 页面上拉触底事件的处理函数
   */
  onReachBottom() {
    console.log('页面触底，加载更多')
    if (this.data.hasMore && !this.data.isLoadingMore) {
      this.loadMoreCourses()
    }
  },

  /**
   * 加载用户信息
   */
  async loadUserInfo() {
    try {
      const userInfo = await authAPI.getProfile()
      this.setData({ userInfo })
    } catch (error) {
      console.error('获取用户信息失败:', error)
    }
  },

  /**
   * 加载课程列表
   */
  async loadCourseList() {
    if (this.data.isLoading) return
    
    this.setData({ isLoading: true })

    try {
      const params = this.buildQueryParams()
      let response
      
      // 根据课件类型调用不同的API
      if (this.data.courseType === 'public') {
        console.log('获取公开课件列表', params)
        response = await courseAPI.getPublicCourses(params)
      } else {
        console.log('获取我的课件列表', params)
        response = await courseAPI.getCourses(params)
      }
      
      console.log('API响应:', response)
      
      // 统一处理响应数据
      const { courses, total } = this.parseResponseData(response)
      
      this.setData({
        courseList: courses,
        'pagination.total': total,
        'pagination.page': 1,
        hasMore: courses.length >= this.data.pagination.limit
      })
    } catch (error) {
      console.error('加载课程列表失败:', error)
      wx.showToast({
        title: '加载失败: ' + (error.message || '网络错误'),
        icon: 'none'
      })
    } finally {
      this.setData({ isLoading: false })
    }
  },

  /**
   * 切换课件类型
   */
  onCourseTypeChange(e) {
    const type = e.currentTarget.dataset.type
    console.log('切换课件类型:', type)
    
    if (this.data.courseType !== type) {
      this.setData({
        courseType: type,
        courseList: [],
        'pagination.page': 1,
        'pagination.total': 0,
        hasMore: true
      })
      
      // 重新加载课件列表
      this.loadCourseList()
    }
  },

  /**
   * 统一解析响应数据
   */
  parseResponseData(response) {
    let courses = []
    let total = 0
    
    if (response && response.courses) {
      courses = response.courses
      total = response.total || 0
    } else if (response && response.data && response.data.courses) {
      courses = response.data.courses
      total = response.data.total || 0
    } else if (Array.isArray(response)) {
      courses = response
      total = response.length
    }
    
    // 处理课程数据
    courses = courses.map(course => {
      if (!course.id && course.ID) {
        course.id = course.ID
      }
      if (course.id && typeof course.id === 'string') {
        const parsedId = parseInt(course.id, 10)
        course.id = isNaN(parsedId) ? null : parsedId
      }
      
      // 过滤description中的重复"基于"内容
      if (course.description) {
        // 如果description以"基于"开头，则不显示，因为我们已经在source_url中显示了
        if (course.description.startsWith('基于')) {
          course.filteredDescription = ''
        } else {
          course.filteredDescription = course.description
        }
      } else {
        course.filteredDescription = ''
      }
      
      return course
    }).filter(course => course.id !== null)
    
    return { courses, total }
  },

  /**
   * 刷新课程列表
   */
  async refreshCourseList() {
    this.setData({ 
      isRefreshing: true,
      'pagination.page': 1
    })

    try {
      const params = this.buildQueryParams()
      let response
      
      // 根据课件类型调用不同的API（与loadCourseList保持一致）
      if (this.data.courseType === 'public') {
        console.log('刷新公开课件列表', params)
        response = await courseAPI.getPublicCourses(params)
      } else {
        console.log('刷新我的课件列表', params)
        response = await courseAPI.getCourses(params)
      }
      
      console.log('刷新课程列表API响应:', response)
      
      // 统一处理响应数据（与loadCourseList保持一致）
      const { courses, total } = this.parseResponseData(response)
      
      this.setData({
        courseList: courses,
        'pagination.total': total,
        'pagination.page': 1,
        hasMore: courses.length >= this.data.pagination.limit
      })
    } catch (error) {
      console.error('刷新课程列表失败:', error)
      wx.showToast({
        title: '刷新失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ isRefreshing: false })
      wx.stopPullDownRefresh()
    }
  },

  /**
   * 加载更多课程
   */
  async loadMoreCourses() {
    if (this.data.isLoadingMore) return
    
    this.setData({ isLoadingMore: true })

    try {
      const nextPage = this.data.pagination.page + 1
      const params = this.buildQueryParams(nextPage)
      const response = await courseAPI.getCourses(params)
      
      console.log('加载更多课程API响应:', response)
      
      // 处理响应数据结构
      let newCourses = []
      
      // 检查响应结构并提取数据
      if (response && response.courses) {
        // 直接返回的数据结构 {courses: [...], total: 55, ...}
        newCourses = response.courses
        console.log('加载更多使用直接返回的数据结构')
      } else if (response && response.data && response.data.courses) {
        // 包装在data中的数据结构 {data: {courses: [...], total: 55, ...}}
        newCourses = response.data.courses
        console.log('加载更多使用包装在data中的数据结构')
      } else if (Array.isArray(response)) {
        // 直接返回数组的情况
        newCourses = response
        console.log('加载更多使用数组数据结构')
      } else {
        console.error('加载更多未知的响应数据结构:', response)
        newCourses = []
      }
      
      console.log('加载更多提取的课程数据:', newCourses)
      console.log('加载更多课程数量:', newCourses.length)
      
      // 确保每个课程都有ID字段
      newCourses = newCourses.map(course => {
        if (!course.id && course.ID) {
          course.id = course.ID
        }
        // 确保ID是数字类型，并过滤无效值
        if (course.id) {
          if (typeof course.id === 'string') {
            const parsedId = parseInt(course.id, 10)
            if (!isNaN(parsedId) && parsedId > 0) {
              course.id = parsedId
            } else {
              console.error('无效的课程ID:', course.id)
              course.id = null
            }
          } else if (typeof course.id === 'number' && course.id > 0) {
            // ID已经是有效的数字
          } else {
            console.error('无效的课程ID:', course.id)
            course.id = null
          }
        } else {
          console.error('课程缺少ID字段:', course)
          course.id = null
        }
        return course
      })
      
      // 过滤掉没有有效ID的课程
      newCourses = newCourses.filter(course => course.id !== null && course.id !== undefined)
      
      const updatedCourseList = [...this.data.courseList, ...newCourses]
      
      this.setData({
        courseList: updatedCourseList,
        'pagination.page': nextPage,
        hasMore: newCourses.length >= this.data.pagination.limit
      })
    } catch (error) {
      console.error('加载更多课程失败:', error)
      wx.showToast({
        title: '加载失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ isLoadingMore: false })
    }
  },

  /**
   * 构建查询参数
   */
  buildQueryParams(page = 1) {
    const { filters, pagination, searchKeyword } = this.data
    const params = {
      page: page,
      page_size: pagination.limit
    }
    
    // 搜索关键词
    if (searchKeyword && searchKeyword.trim()) {
      params.keyword = searchKeyword.trim()
    }
    
    // 状态筛选
    if (filters.status && filters.status !== 'all') {
      params.status = filters.status
    }
    
    // 分类筛选
    if (filters.category && filters.category !== 'all') {
      params.category = filters.category
    }
    
    // 排序参数
    if (filters.sort) {
      params.sort_by = filters.sort
      params.sort_order = filters.order || 'desc'
    }
    
    console.log('构建的查询参数:', params)
    return params
  },

  /**
   * 加载筛选状态
   */
  loadFilterState() {
    try {
      const filterState = wx.getStorageSync('courseFilterState')
      if (filterState) {
        this.setData({
          filters: { ...this.data.filters, ...filterState }
        })
      }
    } catch (error) {
      console.error('加载筛选状态失败:', error)
    }
  },

  /**
   * 保存筛选状态
   */
  saveFilterState() {
    try {
      wx.setStorageSync('courseFilterState', this.data.filters)
    } catch (error) {
      console.error('保存筛选状态失败:', error)
    }
  },

  /**
   * 搜索功能
   */
  onSearchInput(e) {
    const keyword = e.detail.value.trim()
    this.setData({ searchKeyword: keyword })
    
    // 暂时禁用防抖搜索，避免自动触发
    // if (this.data.searchDebounced) {
    //   this.data.searchDebounced()
    // }
  },

  /**
   * 执行搜索
   */
  async performSearch() {
    // 显示搜索功能暂未开放的提示
    wx.showToast({
      title: '搜索功能暂未开放',
      icon: 'none',
      duration: 2000
    })
    
    // 可选：清空搜索关键词
    this.setData({ searchKeyword: '' })
  },

  /**
   * 执行搜索确认
   */
  onSearchConfirm() {
    // 显示搜索功能暂未开放的提示
    wx.showToast({
      title: '搜索功能暂未开放',
      icon: 'none',
      duration: 2000
    })
    
    // 可选：清空搜索关键词
    this.setData({ searchKeyword: '' })
  },

  /**
   * 加载搜索历史
   */
  loadSearchHistory() {
    try {
      const history = wx.getStorageSync('searchHistory') || []
      this.setData({ searchHistory: history })
    } catch (error) {
      console.error('加载搜索历史失败:', error)
    }
  },

  /**
   * 保存搜索历史
   */
  saveSearchHistory(keyword) {
    if (!keyword.trim()) return
    
    try {
      let history = wx.getStorageSync('searchHistory') || []
      
      // 移除重复项
      history = history.filter(item => item !== keyword)
      
      // 添加到开头
      history.unshift(keyword)
      
      // 限制历史记录数量
      if (history.length > 10) {
        history = history.slice(0, 10)
      }
      
      wx.setStorageSync('searchHistory', history)
      this.setData({ searchHistory: history })
    } catch (error) {
      console.error('保存搜索历史失败:', error)
    }
  },

  /**
   * 显示搜索历史
   */
  onSearchFocus() {
    this.setData({ showSearchHistory: true })
  },

  /**
   * 隐藏搜索历史
   */
  onSearchBlur() {
    setTimeout(() => {
      this.setData({ showSearchHistory: false })
    }, 200)
  },

  /**
   * 点击历史项
   */
  onHistoryItemTap(e) {
    const keyword = e.currentTarget.dataset.keyword
    this.setData({ 
      searchKeyword: keyword,
      showSearchHistory: false
    })
    
    // 显示搜索功能暂未开放的提示
    wx.showToast({
      title: '搜索功能暂未开放',
      icon: 'none',
      duration: 2000
    })
  },

  /**
   * 删除历史项
   */
  deleteHistoryItem(e) {
    e.stopPropagation()
    const index = e.currentTarget.dataset.index
    const history = [...this.data.searchHistory]
    history.splice(index, 1)
    
    try {
      wx.setStorageSync('searchHistory', history)
      this.setData({ searchHistory: history })
    } catch (error) {
      console.error('删除搜索历史失败:', error)
    }
  },

  /**
   * 清空搜索历史
   */
  clearSearchHistory() {
    try {
      wx.removeStorageSync('searchHistory')
      this.setData({ searchHistory: [] })
    } catch (error) {
      console.error('清空搜索历史失败:', error)
    }
  },

  /**
   * 清空搜索
   */
  onSearchClear() {
    this.setData({ searchKeyword: '' })
    // 暂时不重新加载课程列表，避免触发搜索
    // this.loadCourseList()
  },

  /**
   * 显示/隐藏筛选器
   */
  toggleFilters() {
    this.setData({
      showFilters: !this.data.showFilters
    })
  },

  /**
   * 状态筛选
   */
  onStatusFilter(e) {
    const status = e.currentTarget.dataset.status
    this.setData({
      'filters.status': status
    })
    this.saveFilterState()
    this.loadCourseList()
  },

  /**
   * 分类筛选
   */
  onCategoryFilter(e) {
    const category = e.currentTarget.dataset.category
    this.setData({
      'filters.category': category
    })
    this.saveFilterState()
    this.loadCourseList()
  },

  /**
   * 排序筛选
   */
  onSortFilter(e) {
    const sort = e.currentTarget.dataset.sort
    const order = this.data.filters.sort === sort && this.data.filters.order === 'desc' ? 'asc' : 'desc'
    
    this.setData({
      'filters.sort': sort,
      'filters.order': order
    })
    this.saveFilterState()
    this.loadCourseList()
  },

  /**
   * 重置筛选
   */
  resetFilters() {
    this.setData({
      filters: {
        status: 'all',
        category: 'all',
        sort: 'created_at',
        order: 'desc'
      }
    })
    this.saveFilterState()
    this.loadCourseList()
  },

  /**
   * 应用筛选
   */
  applyFilters() {
    this.setData({ showFilters: false })
    this.loadCourseList()
  },

  /**
   * 查看课程详情
   */
  viewCourse(e) {
    const courseId = e.currentTarget.dataset.id
    console.log('查看课程，ID:', courseId, '类型:', typeof courseId)
    console.log('事件对象:', e)
    console.log('dataset:', e.currentTarget.dataset)
    
    if (!courseId || courseId === 'undefined' || courseId === undefined) {
      console.error('课程ID无效:', courseId)
      wx.showToast({
        title: '课程ID无效',
        icon: 'none'
      })
      return
    }
    
    // 确保ID是字符串类型
    const idString = String(courseId)
    console.log('跳转到详情页，ID:', idString)
    
    wx.navigateTo({
      url: `/pages/course/detail/detail?id=${idString}`
    })
  },

  /**
   * 创建课程
   */
  createCourse() {
    console.log('创建课程')
    wx.navigateTo({
      url: '/pages/course/create/create'
    })
  },

  /**
   * 编辑课程
   */
  editCourse(e) {
    const courseId = e.currentTarget.dataset.id
    console.log('编辑课程:', courseId)
    
    wx.navigateTo({
      url: `/pages/course/edit/edit?id=${courseId}`
    })
  },

  /**
   * 删除课程
   */
  deleteCourse(e) {
    const courseId = e.currentTarget.dataset.id
    const courseTitle = e.currentTarget.dataset.title
    
    wx.showModal({
      title: '确认删除',
      content: `确定要删除课程"${courseTitle}"吗？此操作无法撤销。`,
      success: async (res) => {
        if (res.confirm) {
          try {
            await courseAPI.deleteCourse(courseId)
            wx.showToast({
              title: '删除成功',
              icon: 'success'
            })
            
            // 重新加载列表
            this.loadCourseList()
          } catch (error) {
            console.error('删除课程失败:', error)
            wx.showToast({
              title: '删除失败: ' + error.message,
              icon: 'none'
            })
          }
        }
      }
    })
  },

  /**
   * 分享课程
   */
  shareCourse(e) {
    const courseId = e.currentTarget.dataset.id
    const courseTitle = e.currentTarget.dataset.title
    
    return {
      title: `分享课程: ${courseTitle}`,
      path: `/pages/course/detail/detail?id=${courseId}`
    }
  },

  /**
   * 获取状态文本
   */
  getStatusText(status) {
    switch (status) {
      case 'generating':
        return '生成中'
      case 'completed':
        return '已完成'
      case 'failed':
        return '失败'
      default:
        return '未知'
    }
  },

  /**
   * 格式化时间 - 显示具体年月日时分
   */
  formatTime(timestamp) {
    if (!timestamp) return ''
    
    try {
      const time = new Date(timestamp)
      
      // 检查时间是否有效
      if (isNaN(time.getTime())) {
        console.error('无效的时间格式:', timestamp)
        return '时间格式错误'
      }
      
      // 显示具体的年月日时分格式
      const year = time.getFullYear()
      const month = String(time.getMonth() + 1).padStart(2, '0')
      const day = String(time.getDate()).padStart(2, '0')
      const hours = String(time.getHours()).padStart(2, '0')
      const minutes = String(time.getMinutes()).padStart(2, '0')
      
      return `${year}-${month}-${day} ${hours}:${minutes}`
    } catch (error) {
      console.error('时间格式化错误:', error, '原始时间:', timestamp)
      return '时间解析错误'
    }
  },

  /**
   * 页面分享
   */
  onShareAppMessage() {
    return {
      title: '我的AI课程集合',
      path: '/pages/course/list/list'
    }
  }
})