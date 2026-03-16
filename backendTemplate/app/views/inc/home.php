<div class='jumbotron jumbotron-fluid'>
<?php if(isset($_SESSION['user_id'])) : ?>
        <div class="container justify-content-center text-center">
        <?php flash('coaching_success');?>
        <?php flash('speaking_success');?>
        <?php flash('therapy_success');?>
        <div class="gallery">
        <h1>Profile</h1>
    <img style="border-radius:50%;" src="asset/img/<?php echo $_SESSION['file_name'] ;?>" alt="">
        <?php endif; ?>
    </div> 
    <div>
            <h4> <?php echo  $_SESSION['user_name'];?></h4>
       
    </div>
        </div>