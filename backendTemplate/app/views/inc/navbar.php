<nav class="navbar navbar-expand-lg bg-warning fixed-top">
  <div class="container-fluid">
  <?php if(isset($_SESSION['user_id'])) : ?> 
    <a class="navbar-brand" href="<?php echo URLROOT;?>"  style = "color:honeydew;">DASHBOARD</a>
    <?php else : ?>
      <a class="navbar-brand" href="<?php echo URLROOT;?>"  style = "color:honeydew;"><?php echo SITENAME;?></a>
      <?php endif; ?>
    <button class="navbar-toggler bg-light" type="button" data-bs-toggle="offcanvas" data-bs-target="#offcanvasNavbar" aria-controls="offcanvasNavbar">
      <span class="navbar-toggler-icon" ></span>
    </button>
    <div class="offcanvas offcanvas-end" tabindex="-1" id="offcanvasNavbar" aria-labelledby="offcanvasNavbarLabel" style = "width:35%; border-radius:10px;" >
      <div class="offcanvas-header" style = "color:honeydew; background-color:goldenrod;">
        <h5 class="offcanvas-title" id="offcanvasNavbarLabel"> Welcome <?php echo  $_SESSION['user_name'];?></h5>
        <button type="button" class="btn-close text-reset bg-light" data-bs-dismiss="offcanvas" aria-label="Close"></button>
      </div>
      <div class="offcanvas-body">
        <ul class="navbar-nav justify-content-end flex-grow-1 pe-3"  style = "color:honeydew;">
          <li class="nav-item">
            <a class="nav-link active" aria-current="page" href="<?php echo URLROOT;?>">Home</a>
          </li>
          <li class="nav-item">
            <a class="nav-link active" href="<?php echo URLROOT;?>/pages/about">About</a>
          </li>
          <li class="nav-item">
            <a class="nav-link active" aria-current="page" href="<?php echo URLROOT;?>/bookme/coachingform">Coaching</a>
          </li>
          <li class="nav-item">
            <a class="nav-link active" href="<?php echo URLROOT;?>/bookme/speakingform">Speaking</a>
          </li>
          <li class="nav-item">
            <a class="nav-link active" href="<?php echo URLROOT;?>/bookme/therapyform">Therapy</a>
          </li>
          <?php if(isset($_SESSION['user_id'])) : ?>
            <li class="nav-item">
            <a class="nav-link active" aria-current="page" href="<?php echo URLROOT;?>/users/logout">Log-Out</a>
          </li
          <?php else : ?>  
          <li class="nav-item">
            <a class="nav-link active" aria-current="page" href="<?php echo URLROOT;?>/users/register">Register</a>
          </li>
          <li class="nav-item">
            <a class="nav-link active" href="<?php echo URLROOT;?>/users/login">Login</a>
          </li>
          <?php endif; ?>
        </ul>
        <!-- <form class="d-flex">
          <input class="form-control me-2" type="search" placeholder="Search" aria-label="Search">
          <button class="btn btn-outline-success" type="submit">Search</button>
        </form> -->
      </div>
    </div>
  </div>
</nav>