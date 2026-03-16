<?php require APPROOT . '/views/inc/header.php'; ?>
<div class="row">
    <div class="col-md-6 mx-auto">
        <div class="card card-body mt-5 ">
            <h2>Create an Account</h2>
            <p>Please fill out this form to register with us</p>
            <form action="<?php echo URLROOT;?>/users/register" method="post" enctype="multipart/form-data">
                <div class="form-group">
                    <label for="name">Name: <sup style="color: red;">*</sup></label>
                    <input type="text" name="name" class="form-control form-control-lg <?php echo (!empty($data['name_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['name'];?>">
                    <span class="invalid-feedback"><?php echo $data['name_err'];?></span>
                </div>
                <div class="form-group">
                    <label for="email">Email: <sup style="color: red;">*</sup></label>
                    <input type="text" name="email" class="form-control form-control-lg <?php echo (!empty($data['email_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['email'];?>">
                    <span class="invalid-feedback"><?php echo $data['email_err'];?></span>
                </div>
                <div class="form-group">
                    <label for="password">Password: <sup style="color: red;">*</sup></label>
                    <input type="password" name="password" class="form-control form-control-lg <?php echo (!empty($data['password_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['password'];?>">
                    <span class="invalid-feedback"><?php echo $data['password_err'];?></span>
                </div> 
                <div class="form-group">
                    <label for="confirm_password">Confirm Password: <sup style="color: red;">*</sup></label>
                    <input type="password" name="confirm_password" class="form-control form-control-lg <?php echo (!empty($data['confirm_password_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['confirm_password'];?>">
                    <span class="invalid-feedback"><?php echo $data['confirm_password_err'];?></span>
                </div> 
                <div class="form-group">
                    <label for="image">Upload an Image: <sup style="color: red;">*</sup></label>
                    <input type="file" accept=".jpg, .jpeg, .png" name="file_name" class="form-control form-control-lg <?php echo (!empty($data['file_name_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['file_name'];?>">
                    <span class="invalid-feedback"><?php echo $data['file_name_err'];?></span>
                </div> <br>
                <div class="row">
                    <div class="col-3">
                        <input type="submit" value="Register" class="btn btn-success btn-block">
                    </div>
                    <div class="col-9">
                        <a href="<?php echo URLROOT;?>/users/login" class="btn btn-light btn-block"> Have an an Account? Login </a>
                    </div>
                </div>
            </form>
        </div>
    </div>
</div>

<?php require APPROOT . '/views/inc/footer.php'; ?>