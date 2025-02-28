/**
 * SEO内容生成系统前端脚本
 */

document.addEventListener('DOMContentLoaded', function () {
    // 移动端导航菜单切换
    setupMobileNav();

    // 搜索功能
    setupSearch();
});

/**
 * 设置移动端导航
 */
function setupMobileNav() {
    const header = document.querySelector('header');
    if (!header) return;

    // 创建汉堡菜单按钮
    const menuButton = document.createElement('button');
    menuButton.className = 'menu-toggle';
    menuButton.innerHTML = '<span></span><span></span><span></span>';
    menuButton.style.display = 'none';

    // 添加到DOM
    const logo = header.querySelector('.logo');
    if (logo) {
        header.querySelector('.container').insertBefore(menuButton, logo.nextSibling);
    }

    // 获取导航菜单
    const nav = header.querySelector('nav ul');
    if (!nav) return;

    // 检查窗口大小并设置菜单状态
    function checkWindowSize() {
        if (window.innerWidth <= 768) {
            menuButton.style.display = 'block';
            nav.classList.remove('active');
        } else {
            menuButton.style.display = 'none';
            nav.style.display = '';
        }
    }

    // 初始检查
    checkWindowSize();

    // 窗口大小变化时重新检查
    window.addEventListener('resize', checkWindowSize);

    // 点击汉堡菜单切换导航显示
    menuButton.addEventListener('click', function () {
        if (nav.classList.contains('active')) {
            nav.classList.remove('active');
        } else {
            nav.classList.add('active');
        }
    });

    // 点击导航链接后关闭菜单
    nav.querySelectorAll('a').forEach(link => {
        link.addEventListener('click', function () {
            if (window.innerWidth <= 768) {
                nav.classList.remove('active');
            }
        });
    });
}

/**
 * 设置搜索功能
 */
function setupSearch() {
    const searchBox = document.querySelector('.search-box');
    if (!searchBox) return;

    const searchInput = searchBox.querySelector('input');
    const searchButton = searchBox.querySelector('button');

    if (!searchInput || !searchButton) return;

    // 搜索函数
    function performSearch() {
        const query = searchInput.value.trim();
        if (query) {
            window.location.href = `/search?q=${encodeURIComponent(query)}`;
        }
    }

    // 点击搜索按钮
    searchButton.addEventListener('click', performSearch);

    // 按回车键搜索
    searchInput.addEventListener('keypress', function (e) {
        if (e.key === 'Enter') {
            performSearch();
        }
    });
}

/**
 * 格式化日期
 * @param {string} dateString - 日期字符串
 * @returns {string} 格式化后的日期
 */
function formatDate(dateString) {
    if (!dateString) return '';

    const date = new Date(dateString);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');

    return `${year}-${month}-${day}`;
}

/**
 * 截断文本
 * @param {string} text - 原始文本
 * @param {number} length - 最大长度
 * @returns {string} 截断后的文本
 */
function truncateText(text, length = 100) {
    if (!text) return '';

    if (text.length <= length) {
        return text;
    }

    return text.substring(0, length) + '...';
}

/**
 * 显示通知
 * @param {string} message - 通知消息
 * @param {string} type - 通知类型 (success, error, info)
 */
function showNotification(message, type = 'info') {
    // 检查是否已有通知元素
    let notification = document.querySelector('.notification');

    // 如果没有，创建一个
    if (!notification) {
        notification = document.createElement('div');
        notification.className = 'notification';
        document.body.appendChild(notification);
    }

    // 设置通知类型和消息
    notification.className = `notification ${type}`;
    notification.textContent = message;

    // 显示通知
    notification.classList.add('show');

    // 3秒后隐藏
    setTimeout(() => {
        notification.classList.remove('show');
    }, 3000);
}

/**
 * 加载更多内容
 * @param {string} url - API URL
 * @param {Function} renderFunction - 渲染函数
 * @param {HTMLElement} container - 容器元素
 * @param {HTMLElement} loadMoreButton - 加载更多按钮
 * @param {number} page - 当前页码
 */
function loadMore(url, renderFunction, container, loadMoreButton, page) {
    // 显示加载中状态
    loadMoreButton.textContent = '加载中...';
    loadMoreButton.disabled = true;

    // 请求数据
    fetch(`${url}&page=${page}`)
        .then(response => response.json())
        .then(data => {
            if (data.code === 200 && data.data.items.length > 0) {
                // 渲染内容
                renderFunction(data.data.items, container, false);

                // 检查是否还有更多内容
                if (data.data.items.length < data.data.page_size || page * data.data.page_size >= data.data.total) {
                    // 没有更多内容
                    loadMoreButton.textContent = '没有更多内容';
                    loadMoreButton.disabled = true;
                } else {
                    // 恢复按钮状态
                    loadMoreButton.textContent = '加载更多';
                    loadMoreButton.disabled = false;

                    // 更新页码
                    loadMoreButton.dataset.page = page + 1;
                }
            } else {
                // 没有更多内容
                loadMoreButton.textContent = '没有更多内容';
                loadMoreButton.disabled = true;
            }
        })
        .catch(error => {
            console.error('加载更多内容失败:', error);
            loadMoreButton.textContent = '加载失败，点击重试';
            loadMoreButton.disabled = false;
        });
} 