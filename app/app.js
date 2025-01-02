document.addEventListener('DOMContentLoaded', async () => {
  const token = localStorage.getItem('token');

  if (token) {
    document.getElementById('auth-section').style.display = 'none';
    document.getElementById('video-section').style.display = 'block';
    await getVideos();
  } else {
    document.getElementById('auth-section').style.display = 'block';
    document.getElementById('video-section').style.display = 'none';
  }
});

document.getElementById('video-draft-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  await createVideoDraft();
});

document.getElementById('login-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  await login();
});

async function createVideoDraft() {
  const title = document.getElementById('video-title').value;
  const description = document.getElementById('video-description').value;

  try {
    const res = await fetch('/api/videos', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({ title, description }),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(`Failed to create video draft: ${data.error}`);
    }

    const videoID = data.id;
    if (videoID) {
      await getVideos();
      await videoStateHandler(videoID);
    }
  } catch (error) {
    alert(`Error: ${error.message}`);
  }
}

async function login() {
  const email = document.getElementById('email').value;
  const password = document.getElementById('password').value;

  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password }),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(`Failed to login: ${data.error}`);
    }

    if (data.token) {
      localStorage.setItem('token', data.token);
      document.getElementById('auth-section').style.display = 'none';
      document.getElementById('video-section').style.display = 'block';
      await getVideos();
    } else {
      alert('Login failed. Please check your credentials.');
    }
  } catch (error) {
    alert(`Error: ${error.message}`);
  }
}

async function signup() {
  const email = document.getElementById('email').value;
  const password = document.getElementById('password').value;

  try {
    const res = await fetch('/api/users', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      const data = await res.json();
      throw new Error(`Failed to create user: ${data.error}`);
    }
    console.log('User created!');
    await login();
  } catch (error) {
    alert(`Error: ${error.message}`);
  }
}

function logout() {
  localStorage.removeItem('token');
  document.getElementById('auth-section').style.display = 'block';
  document.getElementById('video-section').style.display = 'none';
}

function setUploadButtonState(uploading, selector) {
  const uploadBtn = document.getElementById(selector);
  if (uploading) {
    uploadBtn.textContent = 'Uploading...';
    uploadBtn.disabled = true;
    return;
  }
  uploadBtn.textContent = 'Upload';
  uploadBtn.disabled = false;
}

async function uploadThumbnail(videoID) {
  const thumbnailFile = document.getElementById('thumbnail').files[0];
  if (!thumbnailFile) return;

  const formData = new FormData();
  formData.append('thumbnail', thumbnailFile);

  uploadBtnSelector = 'upload-thumbnail-btn';
  setUploadButtonState(true, uploadBtnSelector);

  try {
    const res = await fetch(`/api/thumbnail_upload/${videoID}`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: formData,
    });
    if (!res.ok) {
      const data = await res.json();
      throw new Error(`Failed to upload thumbnail. Error: ${data.error}`);
    }

    await res.json();
    console.log('Thumbnail uploaded!');
    await getVideo(videoID);
  } catch (error) {
    alert(`Error: ${error.message}`);
  }

  setUploadButtonState(false, uploadBtnSelector);
}

async function uploadVideoFile(videoID) {
  const videoFile = document.getElementById('video-file').files[0];
  if (!videoFile) return;

  const formData = new FormData();
  formData.append('video', videoFile);

  uploadBtnSelector = 'upload-video-btn';
  setUploadButtonState(true, uploadBtnSelector);

  try {
    const res = await fetch(`/api/video_upload/${videoID}`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: formData,
    });
    if (!res.ok) {
      const data = await res.json();
      throw new Error(`Failed to upload video file. Error: ${data.error}`);
    }

    console.log('Video uploaded!');
    await getVideo(videoID);
  } catch (error) {
    alert(`Error: ${error.message}`);
  }

  setUploadButtonState(false, uploadBtnSelector);
}

const videoStateHandler = createVideoStateHandler();

async function getVideos() {
  try {
    const res = await fetch('/api/videos', {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
    });
    if (!res.ok) {
      const data = await res.json();
      throw new Error(`Failed to get videos. Error: ${data.error}`);
    }

    const videos = await res.json();
    const videoList = document.getElementById('video-list');
    videoList.innerHTML = '';
    for (const video of videos) {
      const listItem = document.createElement('li');
      listItem.textContent = video.title;
      listItem.onclick = () => videoStateHandler(video.id);
      videoList.appendChild(listItem);
    }
  } catch (error) {
    alert(`Error: ${error.message}`);
  }
}

function createVideoStateHandler() {
  let currentVideoID = null;

  return async function handleVideoClick(videoID) {
    if (currentVideoID !== videoID) {
      currentVideoID = videoID;

      // Reset file input values
      document.getElementById('thumbnail').value = '';
      document.getElementById('video-file').value = '';

      await getVideo(videoID);
    }
  };
}

async function getVideo(videoID) {
  try {
    const res = await fetch(`/api/videos/${videoID}`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
    });
    if (!res.ok) {
      throw new Error('Failed to get video.');
    }

    const video = await res.json();
    viewVideo(video);
  } catch (error) {
    alert(`Error: ${error.message}`);
  }
}

let currentVideo = null;

function viewVideo(video) {
  currentVideo = video;
  document.getElementById('video-display').style.display = 'block';
  document.getElementById('video-title-display').textContent = video.title;
  document.getElementById('video-description-display').textContent = video.description;

  const thumbnailImg = document.getElementById('thumbnail-image');
  if (!video.thumbnail_url) {
    thumbnailImg.style.display = 'none';
  } else {
    thumbnailImg.style.display = 'block';
    thumbnailImg.src = video.thumbnail_url;
  }

  const videoPlayer = document.getElementById('video-player');
  if (videoPlayer) {
    if (!video.video_url) {
      videoPlayer.style.display = 'none';
    } else {
      videoPlayer.style.display = 'block';
      videoPlayer.src = video.video_url;
      videoPlayer.load();
    }
  }
}

async function deleteVideo() {
  if (!currentVideo) {
    alert('No video selected for deletion.');
    return;
  }

  try {
    const res = await fetch(`/api/videos/${currentVideo.id}`, {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
    });
    if (!res.ok) {
      throw new Error('Failed to delete video.');
    }
    alert('Video deleted successfully.');
    document.getElementById('video-display').style.display = 'none';
    await getVideos();
  } catch (error) {
    alert(`Error: ${error.message}`);
  }
}
